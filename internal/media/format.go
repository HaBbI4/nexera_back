package media

import (
	"bytes"
	"fmt"
	"net/http"
)

// Kind описывает тип медиа-файла
type Kind string

const (
	KindPhoto Kind = "photo"
	KindVideo Kind = "video"
)

// Format описывает поддерживаемый формат медиа-файла
type Format struct {
	MIME      string
	Extension string
	Kind      Kind
}

// SniffHeaderSize сколько байт начала файла достаточно прочитать для определения формата
const SniffHeaderSize = 512

// supportedFormats единый реестр поддерживаемых форматов фото/видео.
// Чтобы добавить новый формат, достаточно дописать запись сюда.
var supportedFormats = []Format{
	{MIME: "image/jpeg", Extension: ".jpg", Kind: KindPhoto},
	{MIME: "image/png", Extension: ".png", Kind: KindPhoto},
	{MIME: "image/heic", Extension: ".heic", Kind: KindPhoto},
	{MIME: "video/mp4", Extension: ".mp4", Kind: KindVideo},
	{MIME: "video/quicktime", Extension: ".mov", Kind: KindVideo},
}

// ErrUnsupportedFormat возвращается, когда формат файла не входит в реестр поддерживаемых
type ErrUnsupportedFormat struct {
	Detected string
}

func (e *ErrUnsupportedFormat) Error() string {
	return fmt.Sprintf("unsupported media format: %s", e.Detected)
}

// SupportedFormats возвращает список поддерживаемых форматов (например, для документации API)
func SupportedFormats() []Format {
	result := make([]Format, len(supportedFormats))
	copy(result, supportedFormats)
	return result
}

// lookupByMIME ищет формат в реестре по MIME-типу
func lookupByMIME(mime string) (Format, bool) {
	for _, f := range supportedFormats {
		if f.MIME == mime {
			return f, true
		}
	}
	return Format{}, false
}

// Sniff определяет реальный формат файла по его содержимому (magic bytes),
// а не по расширению или Content-Type, присланным клиентом.
func Sniff(header []byte) (Format, error) {
	if format, ok := detectHEIC(header); ok {
		return format, nil
	}

	mime := http.DetectContentType(header)
	// http.DetectContentType может вернуть уточнённый тип вида "video/mp4; ...",
	// но для наших форматов достаточно точного совпадения без параметров.
	if format, ok := lookupByMIME(trimMIMEParams(mime)); ok {
		return format, nil
	}

	return Format{}, &ErrUnsupportedFormat{Detected: mime}
}

func trimMIMEParams(mime string) string {
	if idx := bytes.IndexByte([]byte(mime), ';'); idx != -1 {
		return mime[:idx]
	}
	return mime
}

// detectHEIC распознаёт HEIC/HEIF по ISO base media file format box "ftyp",
// так как http.DetectContentType из стандартной библиотеки его не поддерживает.
func detectHEIC(header []byte) (Format, bool) {
	if len(header) < 12 {
		return Format{}, false
	}
	if !bytes.Equal(header[4:8], []byte("ftyp")) {
		return Format{}, false
	}

	brand := string(header[8:12])
	switch brand {
	case "heic", "heix", "hevc", "heim", "heis", "hevm", "hevs", "mif1", "msf1":
		f, _ := lookupByMIME("image/heic")
		return f, true
	default:
		return Format{}, false
	}
}
