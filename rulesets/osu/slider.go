package osu

import (
	"github.com/wieku/danser/beatmap/objects"
	"math"
	"github.com/wieku/danser/bmath"
	"sort"
	"github.com/wieku/danser/bmath/difficulty"
)

type objstateS struct {
	buttons  buttonState
	finished bool
	points   []tickpoint
	scored   int64
	slideStart int64
}

type tickpoint struct {
	time       int64
	point      bmath.Vector2d
	scoreGiven HitResult
}

type Slider struct {
	ruleSet           *OsuRuleSet
	hitSlider         *objects.Slider
	players           []*difficultyPlayer
	state             []*objstateS
	fadeStartRelative float64
	lastTime          int64
}

func (slider *Slider) Init(ruleSet *OsuRuleSet, object objects.BaseObject, players []*difficultyPlayer) {
	slider.ruleSet = ruleSet
	slider.hitSlider = object.(*objects.Slider)
	slider.players = players
	slider.state = make([]*objstateS, len(players))

	rSlider := object.(*objects.Slider)

	slider.fadeStartRelative = 100000

	for i, player := range slider.players {
		slider.fadeStartRelative = math.Min(slider.fadeStartRelative, player.diff.Preempt)
		time := int64(math.Max(float64(slider.hitSlider.GetBasicData().StartTime)+float64(slider.hitSlider.GetBasicData().EndTime-slider.hitSlider.GetBasicData().StartTime)/2, float64(slider.hitSlider.GetBasicData().EndTime-36))) //slider ends 36ms before the real end for scoring
		slider.state[i] = new(objstateS)

		for g, point := range rSlider.TickReverse {
			if g < len(rSlider.TickReverse)-1 {
				slider.state[i].points = append(slider.state[i].points, tickpoint{point.Time, point.Pos, HitResults.Slider30})
			}
		}

		slider.state[i].points = append(slider.state[i].points, tickpoint{time, slider.hitSlider.GetPointAt(time), HitResults.Slider30})

		for _, point := range rSlider.TickPoints {
			slider.state[i].points = append(slider.state[i].points, tickpoint{point.Time, point.Pos, HitResults.Slider10})
		}

		sort.Slice(slider.state[i].points, func(g, h int) bool { return slider.state[i].points[g].time < slider.state[i].points[h].time })
		//log.Println(slider.state[i].points)
	}

}

func (slider *Slider) Update(time int64) bool {
	numFinished1 := 0

	for i, player := range slider.players {
		state := slider.state[i]

		yOffset := 0.0
		if player.diff.Mods&difficulty.HardRock > 0 {
			yOffset = slider.hitSlider.GetBasicData().StackOffset.Y*2
		}

		if !state.finished {
			numFinished1++

			if player.cursorLock == -1 {
				state.buttons.Left = player.cursor.LeftButton
				state.buttons.Right = player.cursor.RightButton
			}

			if player.cursorLock == -1 || player.cursorLock == slider.hitSlider.GetBasicData().Number {
				clicked := (!state.buttons.Left && player.cursor.LeftButton) || (!state.buttons.Right && player.cursor.RightButton)

				//log.Println("Huh", time, int64(math.Abs(float64(time - slider.hitSlider.GetBasicData().StartTime))), player.diff.Hit50, clicked, player.cursor.Position.Dst(slider.hitSlider.GetBasicData().StartPos), player.diff.CircleRadius)
				if clicked && player.cursor.Position.Dst(slider.hitSlider.GetBasicData().StartPos.SubS(0, yOffset)) <= player.diff.CircleRadius {
					hit := HitResults.Miss
					combo := ComboResults.Reset

					relative := int64(math.Abs(float64(time - slider.hitSlider.GetBasicData().StartTime)))

					if relative < player.diff.Hit50 {
						hit = HitResults.Slider30
						state.scored++
						combo = ComboResults.Increase
						//log.Println("Huh", relative, player.diff.Hit50)
					} else if relative > int64(player.diff.Preempt-player.diff.FadeIn) {
						hit = HitResults.Ignore
					}

					if hit != HitResults.Ignore {
						if hit == HitResults.Miss {
							hit = HitResults.Ignore
						}
						slider.ruleSet.SendResult(time, player.cursor, slider.hitSlider.GetPosition().X, slider.hitSlider.GetPosition().Y, hit, true, combo)

						player.cursorLock = -1
						state.finished = true
						continue
					}
				}

				player.cursorLock = slider.hitSlider.GetBasicData().Number
			}

			if time > slider.hitSlider.GetBasicData().StartTime+player.diff.Hit50 {
				slider.ruleSet.SendResult(time, player.cursor, slider.hitSlider.GetPosition().X, slider.hitSlider.GetPosition().Y, HitResults.Ignore, true, ComboResults.Reset)
				player.cursorLock = -1
				state.finished = true
				continue
			}

			if player.cursorLock == slider.hitSlider.GetBasicData().Number {
				state.buttons.Left = player.cursor.LeftButton
				state.buttons.Right = player.cursor.RightButton
			}
		}

		if !state.finished {
			continue
		}

		numFinished := 0
		//log.Println("hell?")
		for j, point := range state.points {
			/*if slider.hitSlider.GetBasicData().Number == 728 && j>0 {
				log.Println(time, player.cursorLock, player.cursor.LeftButton, player.cursor.RightButton, state.buttons.Left, state.buttons.Right, player.cursor.Position,player.cursor.Position.Dst(slider.hitSlider.GetPointAt(time)), point.point, slider.hitSlider.GetBasicData().StartPos, slider.hitSlider.GetPartLen(), slider.hitSlider.GetBasicData().EndTime-slider.hitSlider.GetBasicData().StartTime, float64(slider.hitSlider.GetBasicData().StartTime)+float64(slider.hitSlider.GetBasicData().EndTime-slider.hitSlider.GetBasicData().StartTime)/2, float64(slider.hitSlider.GetBasicData().EndTime-36), j, player.diff.CircleRadius, player.diff.CircleRadius*2.4, player.diff.Hit300, player.diff.Hit100, player.diff.Hit50)
			}*/
			//log.Println(time, slider.lastTime, point.time)
			if point.time < slider.lastTime {
				continue
			}
			if numFinished > 0 {
				break
			}
			numFinished++

			//log.Println(point.scoreGiven)
			if j > 0 && time > point.time {
					//log.Println(point.scoreGiven)
					//log.Println(time, slider.lastTime, point, slider.hitSlider.GetBasicData().EndTime)
					if (player.cursor.LeftButton || player.cursor.RightButton) && player.cursor.Position.Dst(/*point.point*/slider.hitSlider.GetPointAt(time).SubS(0, yOffset)) <= player.diff.CircleRadius*2.4 {
						//log.Println(time, point.time, slider.hitSlider.GetBasicData().Number, player.cursor.LeftButton, player.cursor.RightButton, state.buttons.Left, state.buttons.Right, player.cursor.Position.Dst(point.point), player.diff.CircleRadius*2.4)
						slider.ruleSet.SendResult(time, player.cursor, slider.hitSlider.GetPosition().X, slider.hitSlider.GetPosition().Y, point.scoreGiven, true, ComboResults.Increase)
						slider.state[i].scored++
					} else {
						combo := ComboResults.Reset
						if j == len(state.points)-1 {
							combo = ComboResults.Hold
						}
						//log.Println(player.cursor.Position.Dst(point.point) <= player.diff.CircleRadius*2.4, state.buttons.Left || state.buttons.Right, player.cursor.LeftButton || player.cursor.RightButton)
						slider.ruleSet.SendResult(time, player.cursor, slider.hitSlider.GetPosition().X, slider.hitSlider.GetPosition().Y, HitResults.Ignore, true, combo)
					}

				if j == len(state.points)-1 && time >= point.time {
					rate := float64(slider.state[i].scored) / float64(len(state.points))
					hit := HitResults.Miss

					if rate == 1.0 {
						hit = HitResults.Hit300
					} else if rate >= 0.5 {
						hit = HitResults.Hit100
					} else if rate > 0 {
						hit = HitResults.Hit50
					}

					if hit != HitResults.Ignore {
						combo := ComboResults.Hold
						if hit == HitResults.Miss {
							combo = ComboResults.Reset
						}
						//log.Println("Sending miss?")
						slider.ruleSet.SendResult(time, player.cursor, slider.hitSlider.GetPosition().X, slider.hitSlider.GetPosition().Y, hit, false, combo)

						//player.cursorLock = -1
						state.finished = true
					}
				}
			}
		}
		if numFinished > 0 {
			numFinished1++
		}
		state.buttons.Left = player.cursor.LeftButton
		state.buttons.Right = player.cursor.RightButton
	}

	slider.lastTime = time

	return numFinished1 == 0
}

func (slider *Slider) GetFadeTime() int64 {
	return slider.hitSlider.GetBasicData().StartTime - int64(slider.fadeStartRelative)
}