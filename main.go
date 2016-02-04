package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

type Response struct {
	Init     []int
	Sequence []Step
}
type Step struct {
	Tile      int
	Direction string
}
type State struct {
	board [][]int
	blank []int
	sol   string
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("==========START===========")
	init := [][]int{{1, 2, 3}, {4, 0, 5}, {7, 8, 6}}
	goal := [][]int{{1, 2, 3}, {4, 5, 6}, {7, 8, 0}}
	blank := []int{1, 1}
	init, blank = randomPuzzle(init, blank, 40)
	start := copyBoard(init)

	sol := solve(goal, init, blank)
	if len(sol.sol) == 0 {
		fmt.Println("Can't solve")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	fmt.Println("#", init)
	fmt.Println("#Solution : ", sol.sol)
	step, tile := changeBlanktoTile(init, blank, sol.sol)
	solutionJson := returnToFront(start, step, tile)
	//fmt.Println(solutionJson)
	jsonData := struct {
		Json string
	}{
		Json: solutionJson,
	}

	t, _ := template.ParseFiles("app.html")
	t.Execute(w, jsonData)
	//fmt.Fprintf(w, "Hello world!")
}

// ActualCost ( is len(sol) - length of sol string ) is real cost that increase (+1) when move to next state
// Return : return value is HeuristicCost (NOT a actual cost)
func evalFn(goal, now [][]int, actualCost int) int {

	// f(x) = g(x)+h(x) = realCost + heuristicCost
	heuristicCost := heuristicFn(goal, now)
	hFactor := 1
	return actualCost + 1 + hFactor*heuristicCost
}

func heuristicFn(goal, now [][]int) int {
	return h1(goal, now) + h2(goal, now)
}

// Count number of tile that in incorrect position
func h1(goal, now [][]int) int {
	incorrectNo := 0
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if now[i][j] != goal[i][j] {
				incorrectNo++
			}
		}
	}
	return incorrectNo
}

// Sum distance of each tile how far from correct position (Manhattan)
func h2(goal, now [][]int) int {
	sumDistance := 0
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if distance, err := manhattanDistance(goal[i][j], i, j, now); err == nil {
				sumDistance += distance
			} else {
				log.Println("Something Wrong!!!")
			}
		}
	}
	return sumDistance
}

// Manhattan distance
// PS. Actually vert is j, horz is i
func manhattanDistance(targetTile, vertPosition, HorzPosition int, board [][]int) (int, error) {
	// vertDiff+horzDiff
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if board[i][j] == targetTile {
				vertDiff := math.Abs(float64(vertPosition - i))
				horzDiff := math.Abs(float64(HorzPosition - j))
				return int(vertDiff + horzDiff), nil
			}
		}
	}
	return -1, errors.New("Can't find target tile")
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
}

func main() {

	fmt.Println("Hello AI")
	// init := [][]int{{1, 2, 3}, {4, 5, 6}, {7, 8, 0}}
	// goal := [][]int{{0, 2, 3}, {4, 5, 6}, {7, 8, 1}}

	// fmt.Println("h1 ", h1(goal, init))
	// fmt.Println("h2 ", h2(goal, init))

	fmt.Println("Running http://localhost:8000/")
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/favicon.ico", notFoundHandler)
	http.ListenAndServe(":8000", nil)
}
func rndMove(blank []int) string {
	seed := rand.NewSource(time.Now().UnixNano())
	rnd := rand.New(seed)
	var directionRune []rune

	if blank[0] != 0 {
		directionRune = append(directionRune, 'U')
	}
	if blank[0] != 2 {
		directionRune = append(directionRune, 'D')
	}
	if blank[1] != 0 {
		directionRune = append(directionRune, 'L')
	}
	if blank[1] != 2 {
		directionRune = append(directionRune, 'R')
	}
	//fmt.Println("D-RUNE ", directionRune)
	return string(directionRune[rnd.Intn(len(directionRune))])
}
func solve(goal, init [][]int, blank []int) State {
	E := 2.718

	currentState := new(State)
	currentState.board = init
	currentState.sol = ""
	currentState.blank = blank
	seed := rand.NewSource(time.Now().UnixNano())
	rnd := rand.New(seed)

	for t := 1000000; t > 0; t-- {
		if checkIdentical(currentState.board, goal) {
			fmt.Println("---------------SUCCESS------------------")
			fmt.Println("#t=", t)
			return *currentState
		}
		var rndDirection string
		var tmp State
		for {
			rndDirection = rndMove(currentState.blank)
			var err error
			if tmp, err = solveMove(currentState, rndDirection); err == nil {
				break
			}
		}
		//		fmt.Println("T:", t, "@", len(currentState.sol), " Dir:", rndDirection)

		hNext := heuristicFn(goal, tmp.board)
		hParent := heuristicFn(goal, currentState.board)
		deltaH := float64(hNext - hParent)
		//		fmt.Println("deltaH=", hParent, "-", hNext, "=", deltaH)

		if deltaH > 0 {
			//			fmt.Println(">>deltaH pass", deltaH)
			currentState = &tmp
		} else {
			rndProb := rnd.Float64()
			thresholdProb := math.Pow(E, deltaH/float64(t))

			if rndProb < thresholdProb {
				//fmt.Println(">>prob pass", deltaH, ":", rndProb, "<", thresholdProb)
				currentState = &tmp
			} else {
				//fmt.Println("prob fail", deltaH, ":", rndProb, "**", thresholdProb)
			}
		}
	}
	return State{}
}

func solveMove(s *State, dir string) (State, error) {

	board := copyBoard(s.board)
	blank := copyBlank(s.blank)
	board, blank, err := move(board, blank, dir)
	if err != nil {
		//fmt.Println("Can't move!!!!!")
		return State{}, errors.New("Can't move")
	}

	sol := s.sol + dir
	return State{board: board, blank: blank, sol: sol}, err
}

func copyBlank(blank []int) []int {
	return []int{blank[0], blank[1]}
}

func copyBoard(board [][]int) [][]int {
	newBoard := make([][]int, 3)
	newBoard[0] = make([]int, 3)
	newBoard[1] = make([]int, 3)
	newBoard[2] = make([]int, 3)
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			newBoard[i][j] = board[i][j]
		}
	}
	return newBoard
}

func move(board [][]int, blank []int, direction string) ([][]int, []int, error) {
	dir := strings.ToUpper(direction)
	newBoard := copyBoard(board)
	newBlank := copyBlank(blank)
	if canMove(newBlank, dir) {
		switch dir {
		case "U":
			moveU(newBoard, newBlank)
		case "D":
			moveD(newBoard, newBlank)
		case "L":
			moveL(newBoard, newBlank)
		case "R":
			moveR(newBoard, newBlank)
		}
	} else {
		//fmt.Println("can't move.")
		return newBoard, newBlank, errors.New("Can't move")
	}
	return newBoard, newBlank, nil
}

func canMove(blank []int, direction string) bool {
	switch direction {
	case "U":
		if blank[0] <= 0 {
			return false
		}
	case "D":
		if blank[0] >= 2 {
			return false
		}
	case "L":
		if blank[1] <= 0 {
			return false
		}
	case "R":
		if blank[1] >= 2 {
			return false
		}
	default:
		return false
	}
	return true
}

// b is blank
func moveU(board [][]int, b []int) ([][]int, []int) {
	board[b[0]][b[1]] = board[b[0]-1][b[1]]
	board[b[0]-1][b[1]] = 0
	b[0] = b[0] - 1
	return board, b
}
func moveD(board [][]int, b []int) ([][]int, []int) {
	board[b[0]][b[1]] = board[b[0]+1][b[1]]
	board[b[0]+1][b[1]] = 0
	b[0] = b[0] + 1
	return board, b
}
func moveL(board [][]int, b []int) ([][]int, []int) {
	board[b[0]][b[1]] = board[b[0]][b[1]-1]
	board[b[0]][b[1]-1] = 0
	b[1] = b[1] - 1
	return board, b
}
func moveR(board [][]int, b []int) ([][]int, []int) {
	board[b[0]][b[1]] = board[b[0]][b[1]+1]
	board[b[0]][b[1]+1] = 0
	b[1] = b[1] + 1
	return board, b
}

func checkIdentical(b1 [][]int, b2 [][]int) bool {
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if b1[i][j] != b2[i][j] {
				return false
			}
		}
	}
	return true
}

//if number parameter is 0 the randomnumber will generate.
func randomPuzzle(board [][]int, b []int, randomnumber int) ([][]int, []int) {
	//Random seed.
	seed1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(seed1)
	if randomnumber == 0 {
		randomnumber = (r1.Intn(100) + 50)
	}

	//keepRune is an array of random direction sequence.
	var directorRune = []rune("UDLR")
	keepRune := make([]rune, randomnumber)
	for i := range keepRune {
		keepRune[i] = directorRune[r1.Intn(len(directorRune))]
	}
	fmt.Printf("RUNE %c\n", keepRune)
	//Move blank follow by sequence of keepRune.
	for _, direct := range keepRune {
		board, b, _ = move(board, b, string(direct))
	}

	return board, b
}

func print(board [][]int) {
	fmt.Println(board[0])
	fmt.Println(board[1])
	fmt.Println(board[2])
}

func returnToFront(board [][]int, step []string, tile []int) string {
	//fmt.Println("Enter returnToFront")
	initpuzzle := []int{board[0][0], board[0][1], board[0][2],
		board[1][0], board[1][1], board[1][2],
		board[2][0], board[2][1], board[2][2]}

	arrStep := []Step{}
	for i := 0; i < len(step); i++ {
		move := new(Step)
		move.Tile = tile[i]
		move.Direction = step[i]
		arrStep = append(arrStep, *move)
	}
	///fmt.Println(arrStep)

	res := &Response{
		Init:     initpuzzle,
		Sequence: arrStep,
	}
	jsonres, _ := json.Marshal(res)
	//fmt.Println(string(jsonres))
	return string(jsonres)
}

func changeBlanktoTile(board [][]int, b []int, direct string) ([]string, []int) {
	//change blank move to tile move.
	var tile []int
	var move []string
	for i := 0; i < len(direct); i++ {
		switch string(direct[i]) {
		case "U":
			tile = append(tile, board[b[0]-1][b[1]])
			move = append(move, "D")
			board, b = moveU(board, b)
		case "D":
			tile = append(tile, board[b[0]+1][b[1]])
			move = append(move, "U")
			board, b = moveD(board, b)
		case "L":
			tile = append(tile, board[b[0]][b[1]-1])
			move = append(move, "R")
			board, b = moveL(board, b)
		case "R":
			tile = append(tile, board[b[0]][b[1]+1])
			move = append(move, "L")
			board, b = moveR(board, b)
		}
	}
	//fmt.Println(tile)
	//fmt.Println(move)
	return move, tile
}
