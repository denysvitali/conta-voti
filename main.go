package main

import (
	"fmt"
	"github.com/alexflint/go-arg"
	"gocv.io/x/gocv"
	"image"
	"image/color"
	"io/ioutil"
	"log"
	"os"
	"sort"
)

var GreenColor = color.RGBA{
	R: 0,
	G: 255,
	B: 0,
	A: 255,
}

type Cli struct {
	ShowImages bool
	EmptyCard  string
	FilledCard string
	Debug      bool
}

var args struct {
	InputCard  string `arg:"-i"`
	RefCard    string `arg:"-r"`
	ShowImages bool   `arg:"-s"`
	Debug      bool   `arg:"-d"`
}

func main() {
	arg.MustParse(&args)

	if args.InputCard == "" || args.RefCard == "" {
		log.Fatal("missing input cards")
	}

	_, err := os.Open(args.InputCard)
	if err != nil {
		log.Fatal("unable to open input card")
	}

	_, err = os.Open(args.RefCard)
	if err != nil {
		log.Fatal("unable to open ref card")
	}

	c := Cli{
		ShowImages: args.ShowImages,
		EmptyCard:  args.RefCard,
		FilledCard: args.InputCard,
		Debug:      args.Debug,
	}

	votableAreas := c.detectVotable()
	fmt.Printf("votableAreas: %v\n", votableAreas)

	votes := c.detectVotes()
	log.Printf("votes: %v", votes)

	var voteArr []int

	for _, vote := range votes {
		var found = false
		for i, votable := range votableAreas {
			intersect := votable.Intersect(vote)
			//fmt.Printf("intersect (%d): %v\n", i, intersect)
			if !intersect.Empty() {
				// Voted for him!
				log.Printf("voted for %v", i)
				found = true
				voteArr = append(voteArr, i)
				break
			}
		}

		if !found {
			log.Printf("vote not found :(")
		}
	}

	if len(voteArr) > 3 {
		log.Printf("invalid vote")
		os.Exit(-1)
	}
	
	sort.Ints(voteArr)
	for i, v := range voteArr {
		fmt.Printf("%v", v)
		if i < len(voteArr)-1 {
			fmt.Printf(",")
		}
	}

}

func (c Cli) detectVotes() []image.Rectangle {
	emptyColor := gocv.IMRead(c.EmptyCard, gocv.IMReadAnyColor)
	empty := gocv.IMRead(c.EmptyCard, gocv.IMReadGrayScale)
	voted := gocv.IMRead(c.FilledCard, gocv.IMReadGrayScale)

	out := gocv.NewMat()
	gocv.AbsDiff(empty, voted, &out)

	var window *gocv.Window
	if c.ShowImages {
		window = gocv.NewWindow("result")
	}
	c.show(window, out, "absdiff-out")

	cleanedOut := gocv.NewMatWithSize(voted.Rows(), voted.Cols(), voted.Type())
	gocv.Threshold(out, &cleanedOut, 7, 255, gocv.ThresholdBinary)

	if c.ShowImages {
		c.show(window, out, "out")
	}

	kernel := gocv.GetStructuringElement(gocv.MorphEllipse, image.Pt(3, 3))
	gocv.Erode(cleanedOut, &out, kernel)
	gocv.Dilate(out, &cleanedOut, kernel)
	kernel.Close()

	c.show(window, cleanedOut, "cleanedout")
	contours := gocv.FindContours(cleanedOut, gocv.RetrievalExternal, gocv.ChainApproxSimple)

	outputImg := emptyColor.Clone()

	var votes []image.Rectangle

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
			br := gocv.BoundingRect(contour)
			gocv.Rectangle(&outputImg, br, red, 5)
			votes = append(votes, br)
		}
	}

	c.show(window, outputImg, "outputimg")

	return votes
}

type VoteAreas []image.Rectangle

func (v VoteAreas) Len() int {
	return len(v)
}

func (v VoteAreas) Less(i, j int) bool {
	iMin := v[i].Min
	jMin := v[j].Min

	if iMin.Y < jMin.Y {
		return true
	} else if iMin.Y == jMin.Y {
		if iMin.X < jMin.X {
			return true
		}
		return false

	}

	return false
}

func (v VoteAreas) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

var _ sort.Interface = VoteAreas{}

func (c Cli) detectVotable() VoteAreas {
	var window *gocv.Window
	if c.ShowImages {
		window = gocv.NewWindow("detectVotable")
	}

	gray := gocv.IMRead(c.EmptyCard, gocv.IMReadGrayScale)
	gray0 := gray.Clone()
	gocv.Threshold(gray, &gray0, 80, 255, gocv.ThresholdBinary)

	kern := gocv.GetStructuringElement(gocv.MorphEllipse, image.Pt(5, 5))
	gocv.Erode(gray0, &gray, kern)

	var foundRects VoteAreas
	contourMat := gocv.IMRead(c.EmptyCard, gocv.IMReadColor)
	contours := gocv.FindContours(gray, gocv.RetrievalTree, gocv.ChainApproxSimple)

	for _, c := range contours {
		area := gocv.ContourArea(c)
		if area > 500 {
			rect := gocv.BoundingRect(c)
			if rect.Size().X >= 300 && rect.Size().Y > 80 && rect.Size().X < 350 && rect.Size().Y < 110 {
				// Check if rectangle is inside another previously seen rect
				var found = false
				for _, seenRect := range foundRects {
					if rect.In(seenRect) {
						found = true
						break
					}
				}

				if !found {
					foundRects = append(foundRects, rect)
				}
			}
		}
	}

	sort.Sort(foundRects)

	// Draw rects on contourMat
	for i, r := range foundRects {
		gocv.Rectangle(&contourMat, r, GreenColor, 2)
		gocv.PutText(&contourMat, fmt.Sprintf("%d", i), r.Min,
			gocv.FontHersheyPlain,
			5,
			GreenColor,
			2)

	}
	c.show(window, contourMat, "contourMat")

	return foundRects
}

func (c *Cli) show(window *gocv.Window, out gocv.Mat, niceName string) {
	if !c.Debug {
		return
	}
	if window == nil {
		// Window not specified, let's save the image instead
		f, err := ioutil.TempFile(os.TempDir(), "contavoti-"+niceName+"-*.jpg")
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("saving image: %v", f.Name())
		gocv.IMWrite(f.Name(), out)
		return
	}
	window.IMShow(out)
	for {
		if window.WaitKey(5) >= 0 {
			break
		}
	}
}
