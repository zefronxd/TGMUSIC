/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package thumb

import (
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
)

var (
	fontOnce    sync.Once
	ftRegular   *opentype.Font
	ftBold      *opentype.Font
	ftItalic    *opentype.Font
	fontsReady  bool
)

func initFonts() {
	fontOnce.Do(func() {
		var err error
		ftRegular, err = opentype.Parse(goregular.TTF)
		if err != nil {
			return
		}
		ftBold, err = opentype.Parse(gobold.TTF)
		if err != nil {
			return
		}
		ftItalic, err = opentype.Parse(goitalic.TTF)
		if err != nil {
			return
		}
		fontsReady = true
	})
}

// faceOpts returns FaceOptions with standard screen DPI.
func faceOpts(size float64) *opentype.FaceOptions {
	return &opentype.FaceOptions{Size: size, DPI: 96, Hinting: font.HintingFull}
}

// regularFace returns a regular-weight font face at the given point size.
func regularFace(size float64) font.Face {
	initFonts()
	if !fontsReady {
		return nil
	}
	f, _ := opentype.NewFace(ftRegular, faceOpts(size))
	return f
}

// boldFace returns a bold-weight font face at the given point size.
func boldFace(size float64) font.Face {
	initFonts()
	if !fontsReady {
		return nil
	}
	f, _ := opentype.NewFace(ftBold, faceOpts(size))
	return f
}

// italicFace returns an italic font face at the given point size.
func italicFace(size float64) font.Face {
	initFonts()
	if !fontsReady {
		return nil
	}
	f, _ := opentype.NewFace(ftItalic, faceOpts(size))
	return f
}
