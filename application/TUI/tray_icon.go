package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
)

func trayIcon() []byte {
	const size = 32
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	transparent := color.NRGBA{0, 0, 0, 0}
	teal := color.NRGBA{82, 214, 199, 255}
	dark := color.NRGBA{11, 16, 23, 255}
	white := color.NRGBA{232, 238, 245, 255}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.Set(x, y, transparent)
			dx, dy := x-16, y-16
			if dx*dx+dy*dy <= 225 {
				img.Set(x, y, teal)
			}
			if dx*dx+dy*dy <= 169 {
				img.Set(x, y, dark)
			}
		}
	}
	// A compact shield and check mark remain readable at tray size.
	for y := 7; y < 23; y++ {
		span := 9 - (y-7)/3
		for x := 16 - span; x <= 16+span; x++ {
			img.Set(x, y, white)
		}
	}
	for y := 10; y < 20; y++ {
		span := 6 - (y-10)/2
		for x := 16 - span; x <= 16+span; x++ {
			img.Set(x, y, dark)
		}
	}
	for i := 0; i < 5; i++ {
		img.Set(11+i, 15+i, teal)
		img.Set(12+i, 15+i, teal)
	}
	for i := 0; i < 8; i++ {
		img.Set(15+i, 19-i, teal)
		img.Set(15+i, 20-i, teal)
	}
	var pngData bytes.Buffer
	_ = png.Encode(&pngData, img)
	var ico bytes.Buffer
	_ = binary.Write(&ico, binary.LittleEndian, uint16(0))
	_ = binary.Write(&ico, binary.LittleEndian, uint16(1))
	_ = binary.Write(&ico, binary.LittleEndian, uint16(1))
	ico.Write([]byte{size, size, 0, 0})
	_ = binary.Write(&ico, binary.LittleEndian, uint16(1))
	_ = binary.Write(&ico, binary.LittleEndian, uint16(32))
	_ = binary.Write(&ico, binary.LittleEndian, uint32(pngData.Len()))
	_ = binary.Write(&ico, binary.LittleEndian, uint32(22))
	ico.Write(pngData.Bytes())
	return ico.Bytes()
}
