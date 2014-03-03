package main

import (
	"bytes"
	"code.google.com/p/freetype-go/freetype"
	"compress/lzw"
	"encoding/base64"
	"flag"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"log"
	"os"
)

func main() {

	var (
		file    string
		pngfile string
		size    int
		dpi     int
		minset  bool
		packstr bool
	)

	flag.StringVar(&file, "f", "font.ttf", "truetype font filename")
	flag.StringVar(&pngfile, "png", "", "filename for png result")
	flag.IntVar(&size, "pt", 12, "font size")
	flag.IntVar(&dpi, "dpi", 144, "resolution")
	flag.BoolVar(&minset, "min", false, "reduced set of characters")
	flag.BoolVar(&packstr, "lzw", false, "print lzw compressed and base64 encoded string")
	flag.Parse()

	data, err := ioutil.ReadFile(file)

	if err != nil {
		log.Fatal(err)
	}

	font, err := freetype.ParseFont(data)

	if err != nil {
		log.Fatal(err)
	}

	scale := int32(dpi * size)

	fc := freetype.NewContext()
	fc.SetDPI(float64(dpi))
	fc.SetHinting(freetype.FullHinting)
	fc.SetFontSize(float64(size))
	fc.SetFont(font)

	bounds := font.Bounds(scale)
	width := int((bounds.XMax-bounds.XMin)+71) / 72
	height := int((bounds.YMax-bounds.YMin)+71) / 72
	dx := int(-bounds.XMin+71) / 72
	dy := int(bounds.YMax+71) / 72

	offset := 0
	cols, rows := 32, 8

	if minset {
		offset = 32
		cols, rows = 16, 6
	}

	dst := image.NewAlpha(image.Rect(0, 0, width*cols, height*rows))

	fc.SetDst(dst)
	fc.SetSrc(image.White)
	fc.SetClip(dst.Bounds())

	maxAdvance := int32(0)

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			c := y*cols + x + offset
			hm := font.HMetric(scale/72, font.Index(rune(c)))
			if maxAdvance < hm.AdvanceWidth {
				maxAdvance = hm.AdvanceWidth
			}
			p := freetype.Pt(x*width+dx, y*height+dy)
			fc.DrawString(string(c), p)
		}
	}

	fmt.Printf("// %vx%v\n", dst.Rect.Dx(), dst.Rect.Dy())
	fmt.Println("//", width, "x", height, "->", maxAdvance)

	if len(pngfile) > 0 {
		dstfile, err := os.Create(pngfile)
		if err != nil {
			log.Fatal(err)
		}
		defer dstfile.Close()
		png.Encode(dstfile, dst)
	}

	if packstr {
		var b bytes.Buffer

		e := base64.NewEncoder(base64.StdEncoding, &b)
		w := lzw.NewWriter(e, lzw.MSB, 8)
		w.Write(dst.Pix)
		w.Close()
		e.Close()

		nbytes := len(b.Bytes())
		linelen := 72
		rows := nbytes / linelen

		fmt.Println("//", nbytes)

		for r := 0; r < rows; r++ {
			slice := b.Bytes()[r*linelen : (r+1)*linelen]
			fmt.Printf("\"%s\"\n", slice)
		}

		if rows*linelen < nbytes {
			fmt.Printf("\"%s\"\n", b.Bytes()[rows*linelen:])
		}
	}

	if len(pngfile) == 0 && !packstr {
		nbytes := len(dst.Pix)
		cols := 12
		rows := nbytes / cols
		for y := 0; y < rows; y++ {
			for x := 0; x < cols; x++ {
				fmt.Printf(" %#02x,", dst.Pix[y*cols+x])
			}
			fmt.Println("")
		}

		for r := rows * cols; r < nbytes; r++ {
			fmt.Printf(" %#02x,", dst.Pix[r])
		}
		fmt.Println("")
	}
}
