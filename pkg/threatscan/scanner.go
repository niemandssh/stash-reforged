// Package threatscan provides video file threat scanning functionality.
// It searches for security threats in both file metadata and content.
package threatscan

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/stashapp/stash/pkg/ffmpeg"
	"github.com/stashapp/stash/pkg/logger"
	stashExec "github.com/stashapp/stash/pkg/exec"
)

// Threat patterns to detect in metadata and content.
var (
	// Executable extensions that might indicate embedded malware
	executableExtPattern = regexp.MustCompile(`(?i)\.(exe|bat|cmd|ps1|vbs|vbe|js|jse|wsf|wsh|scr|com|msi|dll|jar|deb|rpm|app|swf)\b`)

	// Script/injection patterns
	scriptPattern = regexp.MustCompile(`(?i)(<script|javascript:|vbscript:|data:text/html|onload\s*=|onerror\s*=|onclick\s*=)`)

	// Additional script patterns (eval, exec, DOM manipulation)
	scriptEvalPattern = regexp.MustCompile(`(?i)(\beval\s*\(|document\.write\s*\(|innerHTML\s*=|outerHTML\s*=|exec\s*\(|Function\s*\()`)

	// Shell/command injection patterns
	shellPattern = regexp.MustCompile(`(?i)(/bin/(ba)?sh|cmd\.exe|powershell|wscript\.shell|exec\s*\(\s*["']|system\s*\(\s*["']|Runtime\.getRuntime\(\)\.exec)`)

	// XXE and external entity inclusion
	xxePattern = regexp.MustCompile(`(?i)(<!ENTITY|SYSTEM\s+["']file://|<!DOCTYPE[^>]*\[\s*<!ENTITY)`)

	// Suspicious URL schemes
	suspiciousURLPattern = regexp.MustCompile(`(?i)(file://|data:application/(x-)?(octet-stream|executable)|php://|expect://|dict://|gopher://)`)

	// Base64-like payload indicators (potential encoded malware) - metadata only
	base64PayloadPattern = regexp.MustCompile(`(?i)[A-Za-z0-9+/]{200,}={0,2}`)

	// Crypto miner / malware C2 patterns (specific indicators)
	malwarePattern = regexp.MustCompile(`(?i)(stratum\+tcp://|xmrpool\.|miningpool\.|\.onion\b|stratum\.)`)

	// PHP/web shell patterns
	phpWebShellPattern = regexp.MustCompile(`(?i)(\beval\s*\(\s*\$|base64_decode\s*\(|passthru\s*\(|shell_exec\s*\(|preg_replace\s*\([^)]*\/e\s*\)|assert\s*\(\s*\$|create_function\s*\(|popen\s*\(|proc_open\s*\()`)

	// SSRF - cloud metadata, localhost in URL context
	ssrfPattern = regexp.MustCompile(`(?i)(169\.254\.169\.254|metadata\.google\.internal|file://localhost|http://127\.0\.0\.1|https://127\.0\.0\.1|http://localhost)`)

	// Polyglot - HTML in non-HTML context (e.g. video file or metadata)
	polyglotPattern = regexp.MustCompile(`(?i)<!DOCTYPE\s+html|<html(?:\s|>)`)

	// M3U8/HLS playlist injection - suspicious URI in playlist
	m3u8InjectionPattern = regexp.MustCompile(`(?i)(#EXT-X-KEY|#EXT-X-MAP|#EXT-X-SESSION-DATA).*URI\s*=\s*["']?(file://|javascript:|data:)[^"'\s]*`)

	// Privilege escalation / env hijacking
	envHijackPattern = regexp.MustCompile(`(?i)(LD_PRELOAD|LD_LIBRARY_PATH|DYLD_INSERT_LIBRARIES)\s*=`)

	// Sensitive file access attempts
	sensitiveFilePattern = regexp.MustCompile(`(?i)/(etc/passwd|etc/shadow|etc/sudoers|proc/self/)`)

	// Subtitle exploit patterns (ASS/SSA, WebVTT, SRT)
	assLongOverridePattern = regexp.MustCompile(`(?i)\\{[^}]{200,}\\}`) // Very long ASS override block (potential overflow)
	webVttHtmlPattern      = regexp.MustCompile(`(?i)WEBVTT\b.{0,500}(<script|<iframe|javascript:|on\w+\s*=)`)
	srtHtmlPattern         = regexp.MustCompile(`(?i)\d{2}:\d{2}:\d{2}[,.]\d{3}.{0,200}(<script|<iframe|javascript:|on\w+\s*=)`)

	// DASH MPD URI injection
	dashMpdInjectionPattern = regexp.MustCompile(`(?i)(<BaseURL|<SourceURL|<Initialization)\s*>?\s*(file://|javascript:|data:)[^<"')\s]*`)

	// TTML/DFXP subtitle XXE (XML-based subtitles)
	ttmlDfxpXxePattern = regexp.MustCompile(`(?i)(<tt\s|<dfxp\s|xmlns:tt=).{0,500}(<!ENTITY|SYSTEM\s+["']file://|<!DOCTYPE[^>]*\[\s*<!ENTITY)`)

	// SAMI subtitle (HTML-based, can contain scripts)
	samiHtmlPattern = regexp.MustCompile(`(?i)<SAMI.{0,200}(<script|javascript:|on\w+\s*=)`)

	// VobSub/SUB (binary, but can have embedded paths) - check for path traversal in text
	vobsubPathPattern = regexp.MustCompile(`(?i)\.sub.{0,500}\.\./`)

	// Java deserialization (ObjectInputStream)
	javaSerialMagic = []byte{0xac, 0xed, 0x00, 0x05}

	// Python pickle (protocol 2, 3, 4) - \x80\x02, \x80\x03, \x80\x04
	pythonPickleMagic = []byte{0x80, 0x02}
	pythonPickleMagic3 = []byte{0x80, 0x03}
	pythonPickleMagic4 = []byte{0x80, 0x04}

	// OpenType font (exploit vector in font rendering)
	openTypeMagicOTTO = []byte{0x4f, 0x54, 0x54, 0x4f} // OTTO
	openTypeMagicTrue = []byte{0x74, 0x72, 0x75, 0x65} // true
	openTypeMagicTyp1 = []byte{0x74, 0x79, 0x70, 0x31} // typ1

	// PE executable header (MZ)
	peHeader = []byte{0x4d, 0x5a}

	// ELF executable header
	elfHeader = []byte{0x7f, 0x45, 0x4c, 0x46}

	// Mach-O headers (macOS)
	machOHeader32 = []byte{0xfe, 0xed, 0xfa, 0xce}
	machOHeader64 = []byte{0xfe, 0xed, 0xfa, 0xcf}
	machOHeader64Rev = []byte{0xcf, 0xfa, 0xed, 0xfe}

	// Video format magic bytes (MP4/MOV/M4V - ftyp at offset 4)
	mp4Magic = []byte("ftyp")
	// MKV/WebM - EBML header
	mkvMagic = []byte{0x1a, 0x45, 0xdf, 0xa3}
	// AVI - RIFF....AVI
	aviMagic = []byte{0x52, 0x49, 0x46, 0x46} // RIFF, then AVI at offset 8
	aviSub   = []byte{0x41, 0x56, 0x49, 0x20} // AVI
	// FLV
	flvMagic = []byte{0x46, 0x4c, 0x56} // FLV
	// WMV/ASF
	asfMagic = []byte{0x30, 0x26, 0xb2, 0x75, 0x8e, 0x66, 0xcf, 0x11}
	// RM/RMVB
	rmMagic = []byte{0x2e, 0x52, 0x4d, 0x46} // .RMF
	// MPEG-PS
	mpegMagic = []byte{0x00, 0x00, 0x01}
	// OGG (sometimes used for video)
	oggMagic = []byte{0x4f, 0x67, 0x67, 0x53} // OggS

	// SWF/Flash (exploit vector; FWS/CWS/ZWS at start = file is SWF, not video)
	swfMagicFWS = []byte{0x46, 0x57, 0x53} // FWS
	swfMagicCWS = []byte{0x43, 0x57, 0x53} // CWS
	swfMagicZWS = []byte{0x5a, 0x57, 0x53} // ZWS

	// MP4 atoms known for overflow exploits (CVE-2021-21836 etc.)
	mp4AtomCtts = []byte{0x63, 0x74, 0x74, 0x73}
	mp4AtomStts = []byte{0x73, 0x74, 0x74, 0x73}
	mp4AtomStsc = []byte{0x73, 0x74, 0x73, 0x63}
	mp4AtomCo64 = []byte{0x63, 0x6f, 0x36, 0x34}
	mp4AtomStco = []byte{0x73, 0x74, 0x63, 0x6f}

	// MPEG-TS sync byte
	mpegTSSync = byte(0x47)

	// Max bytes to scan at start/end of file
	maxContentScanBytes = 1024 * 1024 // 1MB
	tailScanBytes       = 512 * 1024  // 512KB at end of file

	// Chunk size for reading file
	scanChunkSize = 64 * 1024 // 64KB

	// Min length for readable strings in content
	minReadableStringLen = 15
)

// Result represents a detected threat.
type Result struct {
	Type    string // "metadata" or "content"
	Message string
}

// Scanner scans video files for security threats.
// FFMpeg is optional; when set, enables steganography LSB analysis on extracted frames.
type Scanner struct {
	FFProbe *ffmpeg.FFProbe
	FFMpeg  *ffmpeg.FFMpeg
}

// NewScanner creates a new threat scanner. FFMpeg can be nil; if set, steganography detection is enabled.
func NewScanner(ffprobe *ffmpeg.FFProbe, ffmpegEncoder *ffmpeg.FFMpeg) *Scanner {
	return &Scanner{FFProbe: ffprobe, FFMpeg: ffmpegEncoder}
}

// Scan performs threat scan on a video file.
// Returns list of detected threats (empty if clean).
func (s *Scanner) Scan(ctx context.Context, filePath string) ([]Result, error) {
	var threats []Result

	// Resolve path for zip-contained files
	resolvedPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	// 1. Scan metadata via ffprobe
	metadataThreats, err := s.scanMetadata(ctx, resolvedPath)
	if err != nil {
		logger.Warnf("Error scanning metadata for %s: %v", filePath, err)
	}
	threats = append(threats, metadataThreats...)

	// 2. Scan file content
	contentThreats, err := s.scanContent(ctx, resolvedPath)
	if err != nil {
		logger.Warnf("Error scanning content for %s: %v", filePath, err)
	}
	threats = append(threats, contentThreats...)

	// 3. Steganography LSB analysis (requires FFMpeg)
	if s.FFMpeg != nil {
		stegoThreats, err := s.scanSteganography(ctx, resolvedPath)
		if err != nil {
			logger.Warnf("Error scanning steganography for %s: %v", filePath, err)
		} else {
			threats = append(threats, stegoThreats...)
		}
	}

	return threats, nil
}

// scanMetadata extracts metadata via ffprobe and searches for threat patterns.
func (s *Scanner) scanMetadata(ctx context.Context, filePath string) ([]Result, error) {
	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		"-show_error",
		"-show_entries", "format_tags=*:stream_tags=*",
		filePath,
	}

	cmd := stashExec.CommandContext(ctx, s.FFProbe.Path(), args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	return scanTextForThreats(string(out), "metadata", true), nil
}

// scanTextForThreats searches text for threat patterns.
// metadataOnly: if true, apply metadata-specific checks (base64) that have high false positive rate in binary content.
func scanTextForThreats(text, source string, metadataOnly bool) []Result {
	var threats []Result

	if matches := executableExtPattern.FindAllStringSubmatch(text, -1); len(matches) > 0 {
		seen := make(map[string]bool)
		var exts []string
		for _, m := range matches {
			if len(m) > 1 && !seen[m[1]] {
				seen[m[1]] = true
				exts = append(exts, "."+strings.ToLower(m[1]))
			}
		}
		if len(exts) > 0 {
			msg := "Suspicious executable extension: " + strings.Join(exts, ", ")
			threats = append(threats, Result{
				Type:    source,
				Message: msg,
			})
		}
	}
	if scriptPattern.MatchString(text) {
		threats = append(threats, Result{
			Type:    source,
			Message: "Script or injection pattern",
		})
	}
	if scriptEvalPattern.MatchString(text) {
		threats = append(threats, Result{
			Type:    source,
			Message: "Code execution pattern (eval/exec)",
		})
	}
	if shellPattern.MatchString(text) {
		threats = append(threats, Result{
			Type:    source,
			Message: "Shell/command execution pattern",
		})
	}
	if xxePattern.MatchString(text) {
		threats = append(threats, Result{
			Type:    source,
			Message: "XXE or external entity inclusion",
		})
	}
	if suspiciousURLPattern.MatchString(text) {
		threats = append(threats, Result{
			Type:    source,
			Message: "Suspicious URL scheme",
		})
	}
	if malwarePattern.MatchString(text) {
		threats = append(threats, Result{
			Type:    source,
			Message: "Crypto miner or C2 infrastructure pattern",
		})
	}
	if phpWebShellPattern.MatchString(text) {
		threats = append(threats, Result{
			Type:    source,
			Message: "PHP/web shell pattern (eval, base64_decode, shell_exec, etc.)",
		})
	}
	if ssrfPattern.MatchString(text) {
		threats = append(threats, Result{
			Type:    source,
			Message: "SSRF pattern (cloud metadata, localhost URL)",
		})
	}
	if polyglotPattern.MatchString(text) {
		threats = append(threats, Result{
			Type:    source,
			Message: "Polyglot HTML/XML in non-document context",
		})
	}
	if m3u8InjectionPattern.MatchString(text) {
		threats = append(threats, Result{
			Type:    source,
			Message: "M3U8/HLS playlist URI injection (file://, javascript:, data:)",
		})
	}
	if envHijackPattern.MatchString(text) {
		threats = append(threats, Result{
			Type:    source,
			Message: "Environment hijacking pattern (LD_PRELOAD, DYLD_INSERT_LIBRARIES)",
		})
	}
	if sensitiveFilePattern.MatchString(text) {
		threats = append(threats, Result{
			Type:    source,
			Message: "Sensitive file path access attempt (/etc/passwd, /proc/self/)",
		})
	}
	if assLongOverridePattern.MatchString(text) {
		threats = append(threats, Result{
			Type:    source,
			Message: "ASS/SSA subtitle: suspiciously long override block (potential parser overflow)",
		})
	}
	if webVttHtmlPattern.MatchString(text) {
		threats = append(threats, Result{
			Type:    source,
			Message: "WebVTT subtitle: embedded HTML/script (XSS vector)",
		})
	}
	if srtHtmlPattern.MatchString(text) {
		threats = append(threats, Result{
			Type:    source,
			Message: "SRT subtitle: embedded HTML/script (XSS vector)",
		})
	}
	if dashMpdInjectionPattern.MatchString(text) {
		threats = append(threats, Result{
			Type:    source,
			Message: "DASH MPD: suspicious URI in BaseURL/SourceURL (file://, javascript:, data:)",
		})
	}
	if ttmlDfxpXxePattern.MatchString(text) {
		threats = append(threats, Result{
			Type:    source,
			Message: "TTML/DFXP subtitle: XXE pattern (external entity, file://)",
		})
	}
	if samiHtmlPattern.MatchString(text) {
		threats = append(threats, Result{
			Type:    source,
			Message: "SAMI subtitle: embedded HTML/script (XSS vector)",
		})
	}
	if vobsubPathPattern.MatchString(text) {
		threats = append(threats, Result{
			Type:    source,
			Message: "VobSub: path traversal pattern (../) in path",
		})
	}
	// Check for unusually long base64-like strings (metadata only - high false positive in video binary)
	if metadataOnly {
		if matches := base64PayloadPattern.FindAllString(text, -1); len(matches) > 0 {
			for _, m := range matches {
				if len(m) > 300 {
					threats = append(threats, Result{
						Type:    source,
						Message: "Large base64-like payload",
					})
					break
				}
			}
		}
	}

	return threats
}

// hasVideoMagic returns true if data starts with a known video container format.
func hasVideoMagic(data []byte) bool {
	if len(data) < 12 {
		return false
	}
	// MP4/MOV/M4V/F4V - ftyp at offset 4
	if len(data) >= 8 && bytes.Equal(data[4:8], mp4Magic) {
		return true
	}
	// MKV/WebM - EBML
	if len(data) >= 4 && bytes.Equal(data[:4], mkvMagic) {
		return true
	}
	// AVI - RIFF....AVI
	if len(data) >= 12 && bytes.Equal(data[:4], aviMagic) && bytes.Equal(data[8:12], aviSub) {
		return true
	}
	// FLV
	if len(data) >= 3 && bytes.Equal(data[:3], flvMagic) {
		return true
	}
	// WMV/ASF
	if len(data) >= 8 && bytes.Equal(data[:8], asfMagic) {
		return true
	}
	// RM/RMVB
	if len(data) >= 4 && bytes.Equal(data[:4], rmMagic) {
		return true
	}
	// MPEG-PS
	if len(data) >= 3 && bytes.Equal(data[:3], mpegMagic) {
		return true
	}
	// OGG
	if len(data) >= 4 && bytes.Equal(data[:4], oggMagic) {
		return true
	}
	return false
}

// hasValidEmbeddedSWF checks if data contains a valid SWF header (magic + version + length).
// Reduces false positives: CWS/ZWS byte sequences often appear by chance in compressed video.
func hasValidEmbeddedSWF(data, magic []byte) bool {
	const minSWFHeaderLen = 8
	const maxSWFSize = 100 * 1024 * 1024 // 100MB
	for i := 0; i <= len(data)-minSWFHeaderLen; i++ {
		if !bytes.Equal(data[i:i+3], magic) {
			continue
		}
		version := data[i+3]
		if version < 1 || version > 21 {
			continue
		}
		length := uint32(data[i+4]) | uint32(data[i+5])<<8 | uint32(data[i+6])<<16 | uint32(data[i+7])<<24
		if length < minSWFHeaderLen || length > maxSWFSize {
			continue
		}
		return true
	}
	return false
}

// hasValidEmbeddedPickle checks for Python pickle with GLOBAL opcode (common in RCE payloads).
func hasValidEmbeddedPickle(data []byte) bool {
	const minLen = 12
	magics := [][]byte{pythonPickleMagic, pythonPickleMagic3, pythonPickleMagic4}
	for _, magic := range magics {
		for i := 0; i <= len(data)-minLen; i++ {
			if !bytes.Equal(data[i:i+2], magic) {
				continue
			}
			// GLOBAL opcode 0x63 ('c') - module\nname\n
			if data[i+2] != 0x63 {
				continue
			}
			// Require newline within next 80 bytes (module name)
			for j := i + 3; j < i+83 && j < len(data); j++ {
				if data[j] == 0x0a {
					return true
				}
			}
		}
	}
	return false
}

// hasValidEmbeddedOpenType checks for embedded OpenType/TrueType font (font parsing exploit vector).
func hasValidEmbeddedOpenType(data []byte) bool {
	const otHeaderLen = 12 // OTTO/true/typ1 (4) + numTables (2) + searchRange (2) + entrySelector (2) + rangeShift (2)
	for i := 0; i <= len(data)-otHeaderLen; i++ {
		if bytes.Equal(data[i:i+4], openTypeMagicOTTO) || bytes.Equal(data[i:i+4], openTypeMagicTrue) ||
			bytes.Equal(data[i:i+4], openTypeMagicTyp1) {
			// Basic sanity: numTables should be reasonable (1-50)
			numTables := int(data[i+4])<<8 | int(data[i+5])
			if numTables >= 1 && numTables <= 50 {
				return true
			}
		}
	}
	return false
}

// hasMP4AtomOverflow checks for MP4 atoms (ctts, stts, stsc, co64, stco) with suspiciously large size.
// CVE-2021-21836 and similar: integer overflow in atom size parsing.
func hasMP4AtomOverflow(data []byte) bool {
	const maxAtomSize = 16 * 1024 * 1024 // 16MB - metadata atoms should be much smaller
	atoms := [][]byte{mp4AtomCtts, mp4AtomStts, mp4AtomStsc, mp4AtomCo64, mp4AtomStco}
	for _, atomType := range atoms {
		for i := 4; i <= len(data)-4; i++ {
			if !bytes.Equal(data[i:i+4], atomType) {
				continue
			}
			size := uint32(data[i-4])<<24 | uint32(data[i-3])<<16 | uint32(data[i-2])<<8 | uint32(data[i-1])
			// size=1 means 64-bit extended size follows; 0xFFFFFFFF or huge size = overflow attempt
			if size == 0xFFFFFFFF || (size > maxAtomSize && size != 1) {
				return true
			}
		}
	}
	return false
}

// hasFLVTagOverflow checks for FLV tags with suspiciously large data size (overflow exploit).
func hasFLVTagOverflow(data []byte) bool {
	const flvHeaderLen = 9  // FLV(3) + version(1) + flags(1) + header size(4)
	const tagHeaderLen = 15 // prev size(4) + type(1) + data size(3) + timestamp(3) + ts ext(1) + stream id(3)
	const maxTagSize = 200 * 1024 * 1024 // 200MB - single tag should not exceed this
	if len(data) < flvHeaderLen+tagHeaderLen || !bytes.Equal(data[:3], flvMagic) {
		return false
	}
	for i := flvHeaderLen; i <= len(data)-11; {
		dataSize := int(data[i+5])<<16 | int(data[i+6])<<8 | int(data[i+7])
		if dataSize > maxTagSize || dataSize < 0 {
			return true
		}
		next := i + tagHeaderLen + dataSize
		if next > len(data) || next <= i {
			break
		}
		i = next
	}
	return false
}

// hasMKVEBMLOverflow checks for MKV/WebM EBML elements with suspiciously large size (overflow exploit).
// EBML uses variable-length integers; 4+ byte size vints with value > 100MB are suspicious.
func hasMKVEBMLOverflow(data []byte, fileSize int64) bool {
	if len(data) < 4 || !bytes.Equal(data[:4], mkvMagic) {
		return false
	}
	const maxElementSize = 200 * 1024 * 1024 // 200MB - metadata elements should be much smaller
	// Scan for 4-byte EBML size vint: first byte 0x08-0x0F (4-byte vint), value = (b&0x0F)<<24 | ...
	for i := 4; i <= len(data)-4; i++ {
		b := data[i]
		if b < 0x08 || b > 0x0F {
			continue
		}
		value := uint64(b&0x0F)<<24 | uint64(data[i+1])<<16 | uint64(data[i+2])<<8 | uint64(data[i+3])
		if value > maxElementSize || (fileSize > 0 && int64(value) > fileSize) {
			return true
		}
	}
	return false
}

// hasAVIRIFFOverflow checks for AVI/RIFF chunks with suspicious size (0xFFFFFFFF or beyond file).
func hasAVIRIFFOverflow(data []byte, fileSize int64) bool {
	if len(data) < 12 || !bytes.Equal(data[:4], aviMagic) || !bytes.Equal(data[8:12], aviSub) {
		return false
	}
	// RIFF: 0-3 RIFF, 4-7 size, 8-11 form type. Chunks start at 12.
	// Chunk: [4:ID][4:size LE][data]. Size excludes 8-byte header.
	for i := 12; i <= len(data)-8; {
		chunkSize := uint32(data[i+4]) | uint32(data[i+5])<<8 | uint32(data[i+6])<<16 | uint32(data[i+7])<<24
		if chunkSize == 0xFFFFFFFF {
			return true
		}
		if fileSize > 0 && int64(chunkSize) > fileSize {
			return true
		}
		if chunkSize > 0x7FFFFFFF {
			return true
		}
		next := i + 8 + int(chunkSize)
		if chunkSize&1 != 0 {
			next++
		}
		if next > len(data) || next <= i {
			break
		}
		i = next
	}
	return false
}

// hasOggPageOverflow checks for Ogg pages with oversized page or segment table.
func hasOggPageOverflow(data []byte) bool {
	const oggPageHeaderLen = 27 // OggS(4) + version(1) + flags(1) + granule(8) + serial(4) + seq(4) + crc(4) + segcount(1)
	const maxPageSize = 65307 // Ogg max: 27 + 255 + 255*255
	if len(data) < oggPageHeaderLen {
		return false
	}
	pos := 0
	for pos <= len(data)-oggPageHeaderLen {
		if !bytes.Equal(data[pos:pos+4], oggMagic) {
			break
		}
		segCount := int(data[pos+26])
		if pos+oggPageHeaderLen+segCount > len(data) {
			break
		}
		pageSize := oggPageHeaderLen + segCount
		for s := 0; s < segCount; s++ {
			pageSize += int(data[pos+oggPageHeaderLen+s])
		}
		if pageSize > maxPageSize {
			return true
		}
		pos += pageSize
		if pos >= len(data) {
			break
		}
	}
	return false
}

// hasASFObjectOverflow checks for ASF/WMV objects with suspicious 64-bit Object Size (overflow exploit).
func hasASFObjectOverflow(data []byte, fileSize int64) bool {
	if len(data) < 24 || !bytes.Equal(data[:8], asfMagic) {
		return false
	}
	const maxObjectSize = 500 * 1024 * 1024 // 500MB
	pos := 0
	for pos <= len(data)-24 {
		// ASF Object: 16 bytes GUID, 8 bytes Object Size (little-endian)
		objSize := uint64(data[pos+16]) | uint64(data[pos+17])<<8 | uint64(data[pos+18])<<16 |
			uint64(data[pos+19])<<24 | uint64(data[pos+20])<<32 | uint64(data[pos+21])<<40 |
			uint64(data[pos+22])<<48 | uint64(data[pos+23])<<56
		if objSize == 0xFFFFFFFFFFFFFFFF || objSize > maxObjectSize {
			return true
		}
		if fileSize > 0 && int64(objSize) > fileSize {
			return true
		}
		if objSize < 24 {
			break
		}
		next := pos + int(objSize)
		if next > len(data) || next <= pos {
			break
		}
		pos = next
	}
	return false
}

// hasRMChunkOverflow checks for RealMedia chunks with suspicious size (overflow exploit).
func hasRMChunkOverflow(data []byte, fileSize int64) bool {
	if len(data) < 8 || !bytes.Equal(data[:4], rmMagic) {
		return false
	}
	// RM chunk: 4-byte ID, 4-byte size (big-endian), data. Parse chunks and check size.
	pos := 0
	for pos <= len(data)-8 {
		chunkSize := uint32(data[pos+4])<<24 | uint32(data[pos+5])<<16 | uint32(data[pos+6])<<8 | uint32(data[pos+7])
		if chunkSize == 0xFFFFFFFF || (fileSize > 0 && int64(chunkSize) > fileSize) {
			return true
		}
		if chunkSize > 0x7FFFFFFF || chunkSize < 8 {
			break
		}
		next := pos + 8 + int(chunkSize)
		if next > len(data) || next <= pos {
			break
		}
		pos = next
		if pos > 256*1024 {
			break
		}
	}
	return false
}

// hasMKVAttachmentExecutable checks for MKV AttachedFiles with executable extensions in filename.
func hasMKVAttachmentExecutable(data []byte) bool {
	if len(data) < 4 || !bytes.Equal(data[:4], mkvMagic) {
		return false
	}
	exts := []string{".exe", ".dll", ".bat", ".cmd", ".ps1", ".vbs", ".jar", ".swf"}
	for _, ext := range exts {
		extBytes := []byte(ext)
		for i := 0; i <= len(data)-len(extBytes); i++ {
			if !bytes.EqualFold(data[i:i+len(extBytes)], extBytes) {
				continue
			}
			// Preceded by printable ASCII or null (filename context)
			if i > 0 {
				b := data[i-1]
				if b != 0 && b != 0x20 && b != '/' && b != '\\' && (b < 0x20 || b > 0x7e) {
					continue
				}
			}
			// Followed by null, space, or end (end of filename)
			if i+len(extBytes) < len(data) {
				b := data[i+len(extBytes)]
				if b != 0 && b != 0x20 && b != '"' && b != '\'' && (b < 0x20 || b > 0x7e) {
					continue
				}
			}
			return true
		}
	}
	return false
}

// hasMPEGTSOverflow checks for MPEG-TS packets with suspicious section length in PSI.
// Only runs when file is actually MPEG-TS (starts with 0x47 sync byte and has aligned packets).
func hasMPEGTSOverflow(data []byte) bool {
	const tsPacketSize = 188
	const maxSectionLength = 1021
	if len(data) < tsPacketSize*3 {
		return false
	}
	// Require MPEG-TS: first bytes must be 0x47 at packet boundaries
	if data[0] != mpegTSSync || data[tsPacketSize] != mpegTSSync || data[tsPacketSize*2] != mpegTSSync {
		return false
	}
	for i := 0; i <= len(data)-tsPacketSize; i += tsPacketSize {
		if data[i] != mpegTSSync {
			continue
		}
		if i+5 >= len(data) {
			break
		}
		payloadStart := 4
		if data[i+3]&0x20 != 0 { // adaptation field
			payloadStart = 5 + int(data[i+4])
			if i+payloadStart >= len(data) {
				continue
			}
		}
		payloadOfs := i + payloadStart
		if payloadOfs+3 >= len(data) {
			continue
		}
		pointer := int(data[payloadOfs])
		if pointer > 0 && payloadOfs+1+pointer+3 <= len(data) {
			sectionLen := int(data[payloadOfs+1+pointer+1]&0x0f)<<8 | int(data[payloadOfs+1+pointer+2])
			if sectionLen > maxSectionLength {
				return true
			}
		}
	}
	return false
}

// hasExecutableMagicAtStart returns true if data starts with executable header (PE/ELF/Mach-O).
func hasExecutableMagicAtStart(data []byte) bool {
	if len(data) < 2 {
		return false
	}
	if bytes.Equal(data[:2], peHeader) {
		return true
	}
	if len(data) >= 4 && bytes.Equal(data[:4], elfHeader) {
		return true
	}
	if len(data) >= 4 && (bytes.Equal(data[:4], machOHeader32) || bytes.Equal(data[:4], machOHeader64) ||
		bytes.Equal(data[:4], machOHeader64Rev)) {
		return true
	}
	return false
}

// scanContent reads file bytes and searches for embedded executable content.
func (s *Scanner) scanContent(ctx context.Context, filePath string) ([]Result, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stating file: %w", err)
	}
	if info.IsDir() {
		return nil, nil
	}

	// Read first bytes for format verification
	headerBuf := make([]byte, 32)
	n, _ := f.Read(headerBuf)
	headerBuf = headerBuf[:n]
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seeking: %w", err)
	}

	// Format mismatch: file claims to be video (by path) but starts with executable
	if hasExecutableMagicAtStart(headerBuf) {
		threats := []Result{{
			Type:    "content",
			Message: "File masquerading as video: starts with executable (PE/ELF/Mach-O)",
		}}
		// Continue with full scan for embedded content
		contentThreats, err := s.scanContentRest(ctx, f, filePath, info.Size(), threats)
		return contentThreats, err
	}

	// Unrecognized format: no video magic and no executable at start
	if len(headerBuf) >= 12 && !hasVideoMagic(headerBuf) {
		msg := "Unrecognized file format: does not match MP4/MKV/AVI/FLV/ASF/MPEG/OGG signatures"
		if len(headerBuf) >= 3 && (bytes.Equal(headerBuf[:3], swfMagicFWS) || bytes.Equal(headerBuf[:3], swfMagicCWS) || bytes.Equal(headerBuf[:3], swfMagicZWS)) {
			msg = "File is SWF/Flash, not a video container (potential exploit vector)"
		}
		threats := []Result{{Type: "content", Message: msg}}
		contentThreats, err := s.scanContentRest(ctx, f, filePath, info.Size(), threats)
		return contentThreats, err
	}

	return s.scanContentRest(ctx, f, filePath, info.Size(), nil)
}

// scanContentRest performs the main content scan (executables, text patterns, tail).
func (s *Scanner) scanContentRest(ctx context.Context, f *os.File, filePath string, fileSize int64, initialThreats []Result) ([]Result, error) {
	var threats []Result
	if initialThreats != nil {
		threats = append(threats, initialThreats...)
	}

	// Scan first portion of file for embedded executables
	scanLimit := maxContentScanBytes
	if fileSize < int64(scanLimit) {
		scanLimit = int(fileSize)
	}
	if scanLimit <= 0 {
		return threats, nil
	}

	reader := bufio.NewReader(io.LimitReader(f, int64(scanLimit)))
	var buf bytes.Buffer

	for {
		select {
		case <-ctx.Done():
			return threats, ctx.Err()
		default:
		}

		chunk := make([]byte, scanChunkSize)
		n, err := reader.Read(chunk)
		if n > 0 {
			buf.Write(chunk[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return threats, err
		}
	}

	data := buf.Bytes()

	// Check for PE executable embedded in file
	for i := 0; i < len(data)-2; i++ {
		if bytes.Equal(data[i:i+2], peHeader) {
			// Verify it looks like PE - e_lfanew at offset 0x3C points to PE signature
			if i+0x40 < len(data) {
				peOffset := int(data[i+0x3C]) | int(data[i+0x3D])<<8 | int(data[i+0x3E])<<16 | int(data[i+0x3F])<<24
				// Sanity check: e_lfanew is typically 0x80-0x200 for normal PE files
				if peOffset > 0 && peOffset < 0x1000 && i+peOffset+4 <= len(data) {
					if bytes.Equal(data[i+peOffset:i+peOffset+4], []byte("PE\x00\x00")) {
						threats = append(threats, Result{
							Type:    "content",
							Message: "Embedded Windows executable (PE) detected",
						})
						break
					}
				}
			}
		}
	}

	// Check for ELF executable
	for i := 0; i <= len(data)-4; i++ {
		if bytes.Equal(data[i:i+4], elfHeader) {
			threats = append(threats, Result{
				Type:    "content",
				Message: "Embedded ELF executable detected",
			})
			break
		}
	}

	// Check for Mach-O executable (macOS)
	for i := 0; i <= len(data)-4; i++ {
		if bytes.Equal(data[i:i+4], machOHeader32) || bytes.Equal(data[i:i+4], machOHeader64) ||
			bytes.Equal(data[i:i+4], machOHeader64Rev) {
			threats = append(threats, Result{
				Type:    "content",
				Message: "Embedded Mach-O executable (macOS) detected",
			})
			break
		}
	}

	// Check for embedded SWF/Flash - must match valid SWF header structure to avoid false positives
	// (CWS/ZWS 3-byte sequences often appear by chance in compressed video data)
	if hasValidEmbeddedSWF(data, swfMagicCWS) || hasValidEmbeddedSWF(data, swfMagicZWS) {
		threats = append(threats, Result{
			Type:    "content",
			Message: "Embedded compressed SWF/Flash detected (potential exploit vector)",
		})
	}

	// Check for Java serialized object (RCE deserialization vector)
	if bytes.Contains(data, javaSerialMagic) {
		threats = append(threats, Result{
			Type:    "content",
			Message: "Embedded Java serialized object detected (deserialization RCE vector)",
		})
	}

	// Check for Python pickle (deserialization RCE - require GLOBAL opcode to reduce false positives)
	if hasValidEmbeddedPickle(data) {
		threats = append(threats, Result{
			Type:    "content",
			Message: "Embedded Python pickle detected (deserialization RCE vector)",
		})
	}

	// Check for embedded OpenType font (font parsing exploit vector)
	if hasValidEmbeddedOpenType(data) {
		threats = append(threats, Result{
			Type:    "content",
			Message: "Embedded OpenType font detected (font parsing exploit vector)",
		})
	}

	// MP4 container: suspicious atom sizes (CVE-2021-21836 integer overflow in ctts/stts/stsc/co64/stco)
	if bytes.Contains(data, mp4Magic) && hasMP4AtomOverflow(data) {
		threats = append(threats, Result{
			Type:    "content",
			Message: "MP4 container: suspicious atom size (potential integer overflow in ctts/stts/stsc/co64/stco)",
		})
	}

	// FLV container: suspicious tag data size (overflow exploit)
	if bytes.Contains(data, flvMagic) && hasFLVTagOverflow(data) {
		threats = append(threats, Result{
			Type:    "content",
			Message: "FLV container: suspicious tag size (potential overflow exploit)",
		})
	}

	// MKV/WebM EBML: suspicious element size (overflow exploit)
	if bytes.Contains(data, mkvMagic) && hasMKVEBMLOverflow(data, fileSize) {
		threats = append(threats, Result{
			Type:    "content",
			Message: "MKV/WebM EBML: suspicious element size (potential overflow exploit)",
		})
	}

	// AVI/RIFF: suspicious chunk size (overflow exploit)
	if bytes.Contains(data, aviMagic) && hasAVIRIFFOverflow(data, fileSize) {
		threats = append(threats, Result{
			Type:    "content",
			Message: "AVI/RIFF container: suspicious chunk size (potential overflow exploit)",
		})
	}

	// Ogg: invalid page or oversized segment table
	if bytes.Contains(data, oggMagic) && hasOggPageOverflow(data) {
		threats = append(threats, Result{
			Type:    "content",
			Message: "Ogg container: invalid page or oversized segment (potential overflow)",
		})
	}

	// MPEG-TS: suspicious section length in PSI
	if hasMPEGTSOverflow(data) {
		threats = append(threats, Result{
			Type:    "content",
			Message: "MPEG-TS: suspicious section length in PSI (potential overflow)",
		})
	}

	// ASF/WMV: suspicious object size (overflow exploit)
	if bytes.Contains(data, asfMagic) && hasASFObjectOverflow(data, fileSize) {
		threats = append(threats, Result{
			Type:    "content",
			Message: "ASF/WMV container: suspicious object size (potential overflow exploit)",
		})
	}

	// RM/RMVB: suspicious chunk size (overflow exploit)
	if bytes.Contains(data, rmMagic) && hasRMChunkOverflow(data, fileSize) {
		threats = append(threats, Result{
			Type:    "content",
			Message: "RealMedia container: suspicious chunk size (potential overflow exploit)",
		})
	}

	// MKV Attachments: executable extension in attached filename
	if bytes.Contains(data, mkvMagic) && hasMKVAttachmentExecutable(data) {
		threats = append(threats, Result{
			Type:    "content",
			Message: "MKV: attached file with executable extension (.exe/.dll/etc)",
		})
	}

	// H.264/HEVC NAL checks removed: 0x00 0x00 0x01 appears often in MP4/MKV (length fields,
	// compressed data), causing false positives. Annex B format (start codes) is used in
	// MPEG-TS/raw streams; MP4/MKV use length-prefix, so these checks are inapplicable.

	// Scan for threat patterns in readable ASCII portions (no base64 - too many false positives)
	textSections := extractReadableStrings(data, minReadableStringLen)
	if len(textSections) > 0 {
		combined := strings.Join(textSections, " ")
		contentThreats := scanTextForThreats(combined, "content", false)
		threats = append(threats, contentThreats...)
	}

	// Scan end of file for appended malware (common steganography technique)
	if fileSize > int64(tailScanBytes+maxContentScanBytes) {
		tailThreats, err := s.scanFileTail(ctx, f, fileSize)
		if err != nil {
			logger.Warnf("Error scanning file tail for %s: %v", filePath, err)
		} else {
			threats = append(threats, tailThreats...)
		}
	}

	return threats, nil
}

// scanFileTail scans the last bytes of a file for appended malware.
func (s *Scanner) scanFileTail(ctx context.Context, f *os.File, fileSize int64) ([]Result, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	offset := fileSize - int64(tailScanBytes)
	if offset < 0 {
		offset = 0
	}

	_, err := f.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("seeking to file tail: %w", err)
	}

	data := make([]byte, tailScanBytes)
	n, err := io.ReadFull(f, data)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, err
	}
	data = data[:n]

	var threats []Result

	// Check for embedded executables in tail
	if bytes.Contains(data, peHeader) {
		for i := 0; i < len(data)-2; i++ {
			if bytes.Equal(data[i:i+2], peHeader) && i+0x40 < len(data) {
				peOffset := int(data[i+0x3C]) | int(data[i+0x3D])<<8 | int(data[i+0x3E])<<16 | int(data[i+0x3F])<<24
				if peOffset > 0 && peOffset < 0x1000 && i+peOffset+4 <= len(data) {
					if bytes.Equal(data[i+peOffset:i+peOffset+4], []byte("PE\x00\x00")) {
						threats = append(threats, Result{
							Type:    "content",
							Message: "Appended Windows executable (PE) at end of file",
						})
						break
					}
				}
			}
		}
	}
	if bytes.Contains(data, elfHeader) {
		threats = append(threats, Result{
			Type:    "content",
			Message: "Appended ELF executable at end of file",
		})
	}
	if bytes.Contains(data, machOHeader32) || bytes.Contains(data, machOHeader64) {
		threats = append(threats, Result{
			Type:    "content",
			Message: "Appended Mach-O executable at end of file",
		})
	}
	if bytes.Contains(data, javaSerialMagic) {
		threats = append(threats, Result{
			Type:    "content",
			Message: "Appended Java serialized object at end of file (deserialization RCE vector)",
		})
	}
	if hasValidEmbeddedPickle(data) {
		threats = append(threats, Result{
			Type:    "content",
			Message: "Appended Python pickle at end of file (deserialization RCE vector)",
		})
	}
	if hasValidEmbeddedOpenType(data) {
		threats = append(threats, Result{
			Type:    "content",
			Message: "Appended OpenType font at end of file (font parsing exploit vector)",
		})
	}

	// Scan for text threats in tail
	textSections := extractReadableStrings(data, minReadableStringLen)
	if len(textSections) > 0 {
		combined := strings.Join(textSections, " ")
		tailThreats := scanTextForThreats(combined, "content", false)
		threats = append(threats, tailThreats...)
	}

	return threats, nil
}

// scanSteganography extracts frames via ffmpeg and analyzes LSB distribution for steganography.
// Flags when LSBs are unusually uniform (potential LSB steganography).
func (s *Scanner) scanSteganography(ctx context.Context, filePath string) ([]Result, error) {
	// Extract 1 frame at 25% into video, scaled to 640x360 for consistent analysis
	args := []string{
		"-v", "error", "-y",
		"-ss", "1", // Seek to 1 second (or start if shorter)
		"-i", filePath,
		"-vframes", "1",
		"-vf", "scale=640:360",
		"-f", "rawvideo",
		"-pix_fmt", "rgb24",
		"-",
	}
	out, err := s.FFMpeg.GenerateOutput(ctx, args, nil)
	if err != nil {
		return nil, err
	}
	// rgb24: 3 bytes per pixel; 640*360 = 230400 pixels; 691200 bytes
	const expectedBytes = 640 * 360 * 3
	if len(out) < expectedBytes/2 {
		return nil, nil // Too little data, skip
	}
	if len(out) > expectedBytes {
		out = out[:expectedBytes]
	}
	// Analyze LSB distribution per channel
	// Steganography often makes LSB distribution unusually uniform (close to 50/50)
	var lsbCount [3][2]int // [R,G,B][0,1]
	for i := 0; i < len(out); i++ {
		ch := i % 3
		lsb := out[i] & 1
		lsbCount[ch][lsb]++
	}
	const minPixels = 10000
	const uniformThreshold = 0.002 // Flag if all channels are within 50% ± 0.2%
	uniform := true
	for ch := 0; ch < 3; ch++ {
		n := lsbCount[ch][0] + lsbCount[ch][1]
		if n < minPixels {
			return nil, nil
		}
		ratio := float64(lsbCount[ch][0]) / float64(n)
		if ratio < 0.5-uniformThreshold || ratio > 0.5+uniformThreshold {
			uniform = false
			break
		}
	}
	if uniform {
		return []Result{{
			Type:    "content",
			Message: "Possible LSB steganography: unusually uniform LSB distribution in video frames",
		}}, nil
	}
	return nil, nil
}

// extractReadableStrings extracts sequences of printable ASCII from binary data.
func extractReadableStrings(data []byte, minLen int) []string {
	var result []string
	var current []byte

	for _, b := range data {
		if b >= 32 && b < 127 {
			current = append(current, b)
		} else {
			if len(current) >= minLen {
				result = append(result, string(current))
			}
			current = nil
		}
	}
	if len(current) >= minLen {
		result = append(result, string(current))
	}
	return result
}

// FormatThreats converts threat results to a string for storage (newline-separated).
func FormatThreats(threats []Result) string {
	if len(threats) == 0 {
		return ""
	}
	parts := make([]string, len(threats))
	for i, t := range threats {
		parts[i] = fmt.Sprintf("[%s] %s", t.Type, t.Message)
	}
	return strings.Join(parts, "\n")
}
