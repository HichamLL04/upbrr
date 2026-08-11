// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
	mkbrr "github.com/autobrr/mkbrr/torrent"

	"github.com/autobrr/upbrr/internal/config"
	"github.com/autobrr/upbrr/internal/languageutil"
	"github.com/autobrr/upbrr/pkg/api"
)

func runTorrentland(ctx context.Context, coreSvc api.Core, sourcePath string, opts cliOptions, cfg config.Config) error {
	fmt.Printf("Torrentland: procesando metadatos para %s...\n", filepath.Base(sourcePath))

	req, err := buildCLIRequest(opts, nil, []string{sourcePath}, 1)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	preview, err := coreSvc.FetchMetadataPreview(ctx, req)
	if err != nil {
		return fmt.Errorf("fetch preview: %w", err)
	}

	meta := preview.PreparedMeta

	// 1. Formatear nombre conforme a la guía de Torrentland
	torrentlandName := buildTorrentlandName(meta)
	if torrentlandName == "" {
		return errors.New("no se pudo generar un nombre conforme a la guía de Torrentland")
	}

	// Conservar extensión original
	ext := filepath.Ext(sourcePath)
	newName := torrentlandName + ext
	dir := filepath.Dir(sourcePath)
	targetPath := filepath.Join(dir, newName)

	fmt.Printf("Nombre generado: %s\n", newName)

	// 2. Crear enlace duro (hardlink) o copiar si falla
	if _, err := os.Stat(targetPath); err == nil {
		fmt.Printf("El archivo de destino ya existe, omitiendo enlace duro: %s\n", targetPath)
	} else {
		fmt.Printf("Creando enlace duro en: %s...\n", targetPath)
		err := os.Link(sourcePath, targetPath)
		if err != nil {
			fmt.Printf("Enlace duro falló (posible dispositivo diferente), copiando archivo...\n")
			if err := copyFile(sourcePath, targetPath); err != nil {
				return fmt.Errorf("copiar archivo: %w", err)
			}
		}
	}

	// 3. Crear el torrent utilizando mkbrr
	torrentPath := filepath.Join(dir, torrentlandName+".torrent")
	fmt.Printf("Generando archivo torrent en: %s...\n", torrentPath)

	maxPieceExp := uint(22) // default max piece size exponent
	_, err = mkbrr.Create(mkbrr.CreateOptions{
		Path:           targetPath,
		Name:           newName,
		OutputPath:     torrentPath,
		IsPrivate:      true,
		MaxPieceLength: &maxPieceExp,
	})
	if err != nil {
		return fmt.Errorf("crear torrent con mkbrr: %w", err)
	}

	// 4. Modificar metadatos del torrent para Torrentland (announce, private y source)
	fmt.Printf("Aplicando etiquetas de Torrentland al archivo torrent...\n")
	mi, err := metainfo.LoadFromFile(torrentPath)
	if err != nil {
		return fmt.Errorf("leer torrent: %w", err)
	}

	mi.Announce = "https://torrentland.li/announce/33ba5e3d88f17f3f7f0271b9840416c6"
	mi.AnnounceList = nil
	mi.UrlList = nil

	info, err := mi.UnmarshalInfo()
	if err != nil {
		return fmt.Errorf("deserializar info torrent: %w", err)
	}

	info.Source = "Torrentland"
	infoBytes, err := bencode.Marshal(info)
	if err != nil {
		return fmt.Errorf("serializar info torrent: %w", err)
	}
	mi.InfoBytes = infoBytes

	f, err := os.OpenFile(torrentPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("abrir torrent para escritura: %w", err)
	}
	defer f.Close()

	err = mi.Write(f)
	if err != nil {
		return fmt.Errorf("guardar torrent modificado: %w", err)
	}

	fmt.Printf("\n¡Proceso completado con éxito!\n")
	fmt.Printf("- Enlace duro: %s\n", targetPath)
	fmt.Printf("- Torrent:     %s\n", torrentPath)

	return nil
}

func buildTorrentlandName(meta api.PreparedMetadata) string {
	parts := []string{}

	// {Título}
	title := ""
	if meta.ExternalMetadata.TMDB != nil && meta.ExternalMetadata.TMDB.Title != "" {
		title = meta.ExternalMetadata.TMDB.Title
	} else if meta.ExternalMetadata.TVDB != nil && meta.ExternalMetadata.TVDB.Name != "" {
		title = meta.ExternalMetadata.TVDB.Name
	} else if meta.Release.Title != "" {
		title = meta.Release.Title
	} else {
		title = filepath.Base(meta.SourcePath)
		if idx := strings.Index(title, "."); idx != -1 {
			title = title[:idx]
		}
	}
	// Limpiar caracteres extraños
	title = strings.ReplaceAll(title, "(", "")
	title = strings.ReplaceAll(title, ")", "")
	title = strings.TrimSpace(title)
	if title != "" {
		parts = append(parts, title)
	}

	// {(Año)/Temporada/Episodio}
	season, episode := meta.SeasonEpisodeWithParsedFallback()
	if season > 0 || episode > 0 {
		if season > 0 && episode > 0 {
			parts = append(parts, fmt.Sprintf("S%02dE%02d", season, episode))
		} else if season > 0 {
			parts = append(parts, fmt.Sprintf("S%02d", season))
		} else if episode > 0 {
			parts = append(parts, fmt.Sprintf("E%02d", episode))
		}
	} else if meta.Release.Year > 0 {
		parts = append(parts, fmt.Sprintf("(%d)", meta.Release.Year))
	}

	// {(Comentario)}
	if meta.Edition != "" {
		parts = append(parts, fmt.Sprintf("(%s)", meta.Edition))
	}

	// {Resolución}
	res := strings.TrimSpace(meta.Release.Resolution)
	if res != "" {
		parts = append(parts, res)
	}

	// {HDR / DV}
	hasDV := meta.WebDV || strings.Contains(strings.ToLower(meta.SourcePath), "dv") || strings.Contains(strings.ToLower(meta.SourcePath), "dolby vision")
	hasHDR := strings.Contains(strings.ToLower(meta.SourcePath), "hdr") || strings.Contains(strings.ToLower(meta.HDR), "hdr")
	if hasDV {
		parts = append(parts, "DV")
	}
	if hasHDR {
		parts = append(parts, "HDR")
	}

	// {Tipo de aporte}
	contrib := ""
	typeVal := strings.ToUpper(strings.TrimSpace(meta.Type))
	if typeVal == "" {
		typeVal = strings.ToUpper(strings.TrimSpace(meta.Release.Type))
	}
	switch typeVal {
	case "REMUX":
		contrib = "BluRay Remux"
	case "ENCODE":
		contrib = "BluRay"
	case "WEBDL", "WEBRIP":
		service := strings.ToUpper(strings.TrimSpace(meta.Service))
		if service != "" {
			switch service {
			case "NF":
				contrib = "NF WEB-DL"
			case "AMZN":
				contrib = "AMZN WEB-DL"
			case "DSNP", "DSN":
				contrib = "DSN+ WEB-DL"
			case "ATVP":
				contrib = "APTV+ WEB-DL"
			default:
				contrib = service + " WEB-DL"
			}
		} else {
			contrib = "WEB-DL"
		}
	case "DISC":
		contrib = "Full BluRay"
	default:
		contrib = "WEB-DL"
	}
	if contrib != "" {
		parts = append(parts, contrib)
	}

	// {Audio}
	audioStr := buildTorrentlandAudioString(meta)
	if audioStr != "" {
		parts = append(parts, audioStr)
	}

	// {Codificación}
	vcodec := strings.TrimSpace(meta.VideoCodec)
	if vcodec != "" {
		// Normalización de codificación según tracker
		vcodecLower := strings.ToLower(vcodec)
		if strings.Contains(vcodecLower, "x264") {
			parts = append(parts, "x264")
		} else if strings.Contains(vcodecLower, "x265") {
			parts = append(parts, "x265")
		} else if strings.Contains(vcodecLower, "hevc") || strings.Contains(vcodecLower, "h.265") || strings.Contains(vcodecLower, "h265") {
			parts = append(parts, "HEVC")
		} else if strings.Contains(vcodecLower, "avc") || strings.Contains(vcodecLower, "h.264") || strings.Contains(vcodecLower, "h264") {
			parts = append(parts, "AVC")
		} else {
			parts = append(parts, vcodec)
		}
	}

	return strings.Join(parts, " ")
}

type tlAudioTrack struct {
	Lang     string
	Format   string
	Channels string
	Atmos    bool
}

func buildTorrentlandAudioString(meta api.PreparedMetadata) string {
	if meta.MediaInfoJSONPath == "" {
		return ""
	}

	data, err := os.ReadFile(meta.MediaInfoJSONPath)
	if err != nil {
		return ""
	}

	var doc struct {
		Media struct {
			Track []map[string]any `json:"track"`
		} `json:"media"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return ""
	}

	tracks := []tlAudioTrack{}
	for _, track := range doc.Media.Track {
		tType, _ := track["@type"].(string)
		if strings.ToLower(tType) != "audio" {
			continue
		}

		rawLang, _ := track["Language"].(string)
		if rawLang == "" {
			rawLang, _ = track["Language_String"].(string)
		}
		lang := canonicalTorrentlandLanguage(rawLang)

		format, _ := track["Format"].(string)
		formatProfile, _ := track["Format_Profile"].(string)
		commercial, _ := track["Format_Commercial"].(string)
		if commercial == "" {
			commercial, _ = track["Format_Commercial_IfAny"].(string)
		}

		normFormat := normalizeTorrentlandAudioFormat(format, formatProfile, commercial)

		channelsVal := ""
		for _, key := range []string{"Channels_Original", "Channels", "Channel_s_"} {
			if v, ok := track[key].(string); ok && v != "" {
				channelsVal = v
				break
			}
			if v, ok := track[key].(float64); ok {
				channelsVal = strconv.FormatFloat(v, 'f', -1, 64)
				break
			}
		}

		normChannels := normalizeTorrentlandChannels(channelsVal)

		atmos := false
		additional, _ := track["Format_AdditionalFeatures"].(string)
		if strings.Contains(strings.ToLower(additional), "atmos") || strings.Contains(strings.ToLower(commercial), "atmos") {
			atmos = true
		}

		tracks = append(tracks, tlAudioTrack{
			Lang:     lang,
			Format:   normFormat,
			Channels: normChannels,
			Atmos:    atmos,
		})
	}

	if len(tracks) == 0 {
		return ""
	}

	// Agrupar pistas que compartan el formato, canales y Atmos
	type audioGroup struct {
		langs []string
		spec  string
	}
	groups := []audioGroup{}

	for _, t := range tracks {
		spec := fmt.Sprintf("%s %s", t.Format, t.Channels)
		if t.Atmos {
			spec += " Atmos"
		}

		found := false
		for i, g := range groups {
			if g.spec == spec {
				// Agregar idioma si no está ya
				dup := false
				for _, l := range g.langs {
					if l == t.Lang {
						dup = true
						break
					}
				}
				if !dup {
					groups[i].langs = append(groups[i].langs, t.Lang)
				}
				found = true
				break
			}
		}
		if !found {
			groups = append(groups, audioGroup{
				langs: []string{t.Lang},
				spec:  spec,
			})
		}
	}

	parts := []string{}
	for _, g := range groups {
		langsStr := strings.Join(g.langs, "-")
		parts = append(parts, fmt.Sprintf("%s %s", langsStr, g.spec))
	}

	return strings.Join(parts, " ")
}

func canonicalTorrentlandLanguage(val string) string {
	norm := strings.ToLower(languageutil.NormalizeLanguageDisplay(val))
	if norm == "" {
		norm = strings.ToLower(strings.TrimSpace(val))
	}
	switch norm {
	case "spanish", "castilian", "castellano", "spa", "es", "es-es":
		return "ESP"
	case "latino", "latin america", "es-419", "es-mx", "es-ar":
		return "LAT"
	case "english", "eng", "en":
		return "ENG"
	case "japanese", "jap", "ja":
		return "JAP"
	case "catalan", "ca", "cat":
		return "CAT"
	case "basque", "eu", "eus":
		return "EUS"
	case "galician", "gl", "gal":
		return "GAL"
	case "french", "fr", "fre":
		return "FRE"
	case "german", "de", "ger":
		return "GER"
	case "italian", "it", "ita":
		return "ITA"
	case "portuguese", "pt", "por":
		return "POR"
	default:
		// Tomar las primeras 3 letras del nombre normalizado en inglés
		if len(norm) > 3 {
			return strings.ToUpper(norm[:3])
		}
		return strings.ToUpper(norm)
	}
}

func normalizeTorrentlandAudioFormat(format, profile, commercial string) string {
	formatLower := strings.ToLower(format)
	commLower := strings.ToLower(commercial)
	profLower := strings.ToLower(profile)

	switch {
	case strings.Contains(commLower, "dts-hd master audio") || strings.Contains(profLower, "ma"):
		return "DTS-HD MA"
	case strings.Contains(commLower, "dts-hd high") || strings.Contains(profLower, "hra"):
		return "DTS-HD"
	case strings.Contains(formatLower, "dts"):
		return "DTS"
	case strings.Contains(commLower, "dolby truehd") || strings.Contains(formatLower, "mlp fba"):
		return "TrueHD"
	case strings.Contains(commLower, "dolby digital plus") || strings.Contains(formatLower, "e-ac-3") || strings.Contains(formatLower, "enhanced ac-3"):
		return "DD+"
	case strings.Contains(commLower, "dolby digital") || strings.Contains(formatLower, "ac-3"):
		return "DD"
	case strings.Contains(formatLower, "flac"):
		return "FLAC"
	case strings.Contains(formatLower, "pcm") || strings.Contains(formatLower, "lpcm"):
		return "LPCM"
	case strings.Contains(formatLower, "aac"):
		return "AAC"
	case strings.Contains(formatLower, "opus"):
		return "OPUS"
	case strings.Contains(formatLower, "mp3"):
		return "MP3"
	default:
		return strings.ToUpper(format)
	}
}

func normalizeTorrentlandChannels(val string) string {
	clean := strings.ToLower(strings.TrimSpace(val))
	if strings.Contains(clean, "8") || strings.Contains(clean, "7.1") {
		return "7.1"
	}
	if strings.Contains(clean, "7") || strings.Contains(clean, "6.1") {
		return "6.1"
	}
	if strings.Contains(clean, "6") || strings.Contains(clean, "5.1") {
		return "5.1"
	}
	if strings.Contains(clean, "2") || strings.Contains(clean, "2.0") {
		return "2.0"
	}
	if strings.Contains(clean, "1") || strings.Contains(clean, "1.0") {
		return "1.0"
	}
	return "2.0"
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Sync()
}
