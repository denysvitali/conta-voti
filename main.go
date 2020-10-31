package main

import (
	"fmt"
	"gocv.io/x/gocv"
	"image/color"
	"image"
)

var GreenColor = color.RGBA{
	R: 0,
	G: 255,
	B: 0,
	A: 255,
}

func main(){
	vuota := "/home/dvitali/Downloads/photo_2020-10-31_11-28-15.jpg"
	piena := "/home/dvitali/Downloads/photo_2020-10-31_12-43-00.jpg"
	
	votableAreas := detectVotable(vuota)
	fmt.Printf("votable areas: %v\n", votableAreas)
	
	emptyColor := gocv.IMRead(vuota, gocv.IMReadAnyColor)
	empty := gocv.IMRead(vuota, gocv.IMReadGrayScale)
	voted := gocv.IMRead(piena, gocv.IMReadGrayScale)

	out := gocv.NewMat()
	gocv.AbsDiff(empty, voted, &out)

	window := gocv.NewWindow("result")
	show(window, out)

	cleanedOut := gocv.NewMatWithSize(voted.Rows(), voted.Cols(), voted.Type())
	gocv.Threshold(out, &cleanedOut, 7, 255, gocv.ThresholdBinary)
	show(window, out)

	kernel := gocv.GetStructuringElement(gocv.MorphEllipse, image.Pt(3, 3))
	gocv.Erode(cleanedOut , &out, kernel)
	gocv.Dilate(out , &cleanedOut, kernel)
	kernel.Close()
	
	show(window, cleanedOut)
	contours := gocv.FindContours(cleanedOut, gocv.RetrievalExternal, gocv.ChainApproxSimple)
	
	outputImg := emptyColor.Clone()

	red := color.RGBA{
		R: 255,
		G: 0,
		B: 0,
		A: 255,
	}
	for _, contour := range contours {
		approxCurve := gocv.ApproxPolyDP(contour, 6, false)
		area := gocv.ContourArea(approxCurve)
		if area > 20 {
			fmt.Printf("area: %v\n", area)
			br := gocv.BoundingRect(contour)
			gocv.Rectangle(&outputImg, br, red, 5)
			fmt.Printf("")
		}
	}

	show(window, outputImg)
	
}

func detectVotable(scheda string) []image.Rectangle {
	window := gocv.NewWindow("detectVotable")

	gray := gocv.IMRead(scheda, gocv.IMReadGrayScale)
	gray0 := gray.Clone()
	gocv.Threshold(gray, &gray0, 80, 255, gocv.ThresholdBinary)

	kern := gocv.GetStructuringElement(gocv.MorphEllipse, image.Pt(5, 5))
	gocv.Erode(gray0, &gray, kern)

	var foundRects []image.Rectangle
	contourMat := gocv.IMRead(scheda, gocv.IMReadColor)
	contours := gocv.FindContours(gray, gocv.RetrievalTree, gocv.ChainApproxSimple)

	for _, c := range contours {
		area := gocv.ContourArea(c)
		fmt.Printf("area: %v\n", area)
		
		if area > 500 {
			rect := gocv.BoundingRect(c)
			if rect.Size().X >= 300 && rect.Size().Y > 80 {
				fmt.Printf("size: X = %v, Y = %v\n", rect.Size().X, rect.Size().Y)
				gocv.Rectangle(&contourMat, rect, GreenColor, 2)
				foundRects = append(foundRects, rect)
			}
		}
	}
	
	show(window, contourMat)
	return foundRects
}

func show(window *gocv.Window, out gocv.Mat) {
	window.IMShow(out)
	for ;; {
		if window.WaitKey(5) >= 0 {
			break
		}
	}
}
