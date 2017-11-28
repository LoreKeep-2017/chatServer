package chat

import (
	"bytes"
	"errors"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
)

func convertString(str string, format string, dest *os.File) error {
	bytesData := bytes.NewReader([]byte(str))
	switch format {
	case "png":
		im, err := png.Decode(bytesData)
		if err != nil {
			return err
		}
		return png.Encode(dest, im)
	case "jpeg":
		var opt jpeg.Options
		opt.Quality = 80
		im, err := jpeg.Decode(bytesData)
		if err != nil {
			return err
		}
		return jpeg.Encode(dest, im, &opt)
	case "gif":
		var opt gif.Options
		im, err := gif.Decode(bytesData)
		if err != nil {
			return err
		}
		return gif.Encode(dest, im, &opt)
	}
	return errors.New("format only jpeg/gif/png")

	// switch strings.TrimSuffix(str[5:coI], ";base64") {
	// case "image/png":
	// 	pngI, err := png.Decode(res)
	// 	// ...
	// case "image/jpeg":
	// 	jpgI, err := jpeg.Decode(res)
	// 	// ...
	// }
}
