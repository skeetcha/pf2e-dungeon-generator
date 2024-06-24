package main

import (
	"fmt"
	"maps"
	"math"
	"math/rand"
	"sort"
	"time"
)

// # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
// configuration

var dungeonLayout = map[string][][]int{
	"Box":   {{1, 1, 1}, {1, 0, 1}, {1, 1, 1}},
	"Cross": {{0, 1, 0}, {1, 1, 1}, {0, 1, 0}},
}

var corridorLayout = map[string]int{
	"Labyrinth": 0,
	"Bent":      50,
	"Straight":  100,
}

var mapStyle = map[string]map[string]string{
	"Standard": {
		"fill":      "000000",
		"open":      "FFFFFF",
		"open_grid": "CCCCCC",
	},
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
// cell bits

const NOTHING = 0x00000000

const BLOCKED = 0x00000001
const ROOM = 0x00000002
const CORRIDOR = 0x00000004

// 0x00000008
const PERIMETER = 0x00000010
const ENTRANCE = 0x00000020
const ROOM_ID = 0x0000FFC0

const ARCH = 0x00010000
const DOOR = 0x00020000
const LOCKED = 0x00040000
const TRAPPED = 0x00080000
const SECRET = 0x00100000
const PORTC = 0x00200000
const STAIR_DN = 0x00400000
const STAIR_UP = 0x00800000

const LABEL = 0xFF000000

const OPENSPACE = ROOM | CORRIDOR
const DOORSPACE = ARCH | DOOR | LOCKED | TRAPPED | SECRET | PORTC
const ESPACE = ENTRANCE | DOORSPACE | 0xFF000000
const STAIRS = STAIR_DN | STAIR_UP

const BLOCK_ROOM = BLOCKED | ROOM
const BLOCK_CORR = BLOCKED | PERIMETER | CORRIDOR
const BLOCK_DOOR = BLOCKED | DOORSPACE

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
// directions

var di = map[string]int{"north": -1, "south": 1, "west": 0, "east": 0}
var dj = map[string]int{"north": 0, "south": 0, "west": -1, "east": 1}
var djDirs = KeysFromMap(dj)

var opposite = map[string]string{
	"north": "south",
	"south": "north",
	"west":  "east",
	"east":  "west",
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
// stairs

var stairEnd = map[string]map[string][][]int{
	"north": {
		"walled":   {{1, -1}, {0, -1}, {-1, -1}, {-1, 0}, {-1, 1}, {0, 1}, {1, 1}},
		"corridor": {{0, 0}, {1, 0}, {2, 0}},
		"stair":    {{0, 0}},
		"next":     {{1, 0}},
	},
	"south": {
		"walled":   {{-1, -1}, {0, -1}, {1, -1}, {1, 0}, {1, 1}, {0, 1}, {-1, 1}},
		"corridor": {{0, 0}, {-1, 0}, {-2, 0}},
		"stair":    {{0, 0}},
		"next":     {{-1, 0}},
	},
	"west": {
		"walled":   {{-1, 1}, {-1, 0}, {-1, -1}, {0, -1}, {1, -1}, {1, 0}, {1, 1}},
		"corridor": {{0, 0}, {0, 1}, {0, 2}},
		"stair":    {{0, 0}},
		"next":     {{0, 1}},
	},
	"east": {
		"walled":   {{-1, -1}, {-1, 0}, {-1, 1}, {0, 1}, {1, 1}, {1, 0}, {1, -1}},
		"corridor": {{0, 0}, {0, -1}, {0, -2}},
		"stair":    {{0, 0}},
		"next":     {{0, -1}},
	},
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
// cleaning

var closeEnd = map[string]map[string][][]int{
	"north": {
		"walled":  {{0, -1}, {1, -1}, {1, 0}, {1, 1}, {0, 1}},
		"close":   {{0, 0}},
		"recurse": {{-1, 0}},
	},
	"south": {
		"walled":  {{0, -1}, {1, -1}, {-1, 0}, {-1, 1}, {0, 1}},
		"close":   {{0, 0}},
		"recurse": {{1, 0}},
	},
	"west": {
		"walled":  {{-1, 0}, {-1, 1}, {0, 1}, {1, 1}, {1, 0}},
		"close":   {{0, 0}},
		"recurse": {{0, -1}},
	},
	"east": {
		"walled":  {{-1, 0}, {-1, -1}, {0, -1}, {1, -1}, {1, 0}},
		"close":   {{0, 0}},
		"recurse": {{0, 1}},
	},
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
// imaging

var colorChain = map[string]string{
	"door":  "fill",
	"label": "fill",
	"stair": "wall",
	"wall":  "fill",
	"fill":  "black",
}

// # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
// showtime
var opts map[string]any
var dungeon map[string]any

func main() {
	sort.Strings(djDirs)
	opts = getOpts()
	createDungeon(&dungeon, opts)
	imageDungeon(dungeon)
	fmt.Println("Hello world")
}

func getOpts() map[string]any {
	return map[string]any{
		"seed":            time.Now().Unix(),
		"n_rows":          39,
		"n_cols":          39,
		"dungeon_layout":  "None",
		"room_min":        3,
		"room_max":        9,
		"room_layout":     "Scattered",
		"corridor_layout": "Bent",
		"remove_deadends": 50,
		"add_stairs":      2,
		"map_style":       "Standard",
		"cell_size":       18,
	}
}

func createDungeon(dungeon *map[string]any, opts map[string]any) {
	*dungeon = maps.Clone(opts)
	(*dungeon)["n_i"] = int((*dungeon)["n_rows"].(int) / 2)
	(*dungeon)["n_j"] = int((*dungeon)["n_cols"].(int) / 2)
	(*dungeon)["max_row"] = (*dungeon)["n_rows"].(int) - 1
	(*dungeon)["max_col"] = (*dungeon)["n_cols"].(int) - 1
	(*dungeon)["n_rooms"] = 0
	(*dungeon)["room_base"] = int(((*dungeon)["room_min"].(int) + 1) / 2)
	(*dungeon)["room_radix"] = int(((*dungeon)["room_max"].(int)-(*dungeon)["room_min"].(int))/2) + 1
	initCells(dungeon)
	emplaceRooms(dungeon)
	openRooms(dungeon)
	labelRooms(dungeon)
	corridors(dungeon)
	emplaceStairs(dungeon)
	cleanDungeon(dungeon)
}

func initCells(dungeon *map[string]any) {
	(*dungeon)["cell"] = make([][]any, (*dungeon)["n_rows"].(int))

	for r := 0; r <= (*dungeon)["n_rows"].(int); r++ {
		(*dungeon)["cell"].([][]any)[r] = make([]any, (*dungeon)["n_cols"].(int))

		for c := 0; c <= (*dungeon)["n_cols"].(int); c++ {
			(*dungeon)["cell"].([][]any)[r][c] = nil
		}
	}

	(*dungeon)["random"] = rand.New(rand.NewSource((*dungeon)["seed"].(int64)))

	if mask := dungeonLayout[(*dungeon)["dungeon_layout"].(string)]; mask != nil {
		maskCells(dungeon, mask)
	} else if (*dungeon)["dungeon_layout"].(string) == "Round" {
		roundMask(dungeon)
	}
}

func maskCells(dungeon *map[string]any, mask [][]int) {
	r_x := float32(len(mask)) / float32((*dungeon)["n_rows"].(int)+1)
	c_x := float32(len(mask[0])) / float32((*dungeon)["n_cols"].(int)+1)

	for r := 0; r <= (*dungeon)["n_rows"].(int); r++ {
		for c := 0; c <= (*dungeon)["n_cols"].(int); c++ {
			if mask[int(float32(r)*r_x)][int(float32(c)*c_x)] == 0 {
				(*dungeon)["cell"].([][]any)[r][c] = BLOCKED
			}
		}
	}
}

func roundMask(dungeon *map[string]any) {
	center_r := (*dungeon)["n_rows"].(int) / 2
	center_c := (*dungeon)["n_cols"].(int) / 2

	for r := 0; r <= (*dungeon)["n_rows"].(int); r++ {
		for c := 0; c <= (*dungeon)["n_cols"].(int); c++ {
			d := math.Sqrt((math.Pow(float64(r-center_r), 2) + (math.Pow(float64(c-center_c), 2))))

			if d <= float64(center_c) {
				(*dungeon)["cell"].([][]any)[r][c] = BLOCKED
			}
		}
	}
}

func emplaceRooms(dungeon *map[string]any) {
	if (*dungeon)["room_layout"].(string) == "Packed" {
		packRooms(dungeon)
	} else {
		scatterRooms(dungeon)
	}
}

func packRooms(dungeon *map[string]any) {
	for i := 0; i < (*dungeon)["n_i"].(int); i++ {
		r := (i * 2) + 1

		for j := 0; j < (*dungeon)["n_j"].(int); j++ {
			c := (j * 2) + 1

			if ((*dungeon)["cell"].([][]any)[r][c].(int) & ROOM) != 0 {
				continue
			}

			if ((i == 0) || (j == 0)) && ((*(*dungeon)["rand"].(*rand.Rand)).Intn(2) != 0) {
				continue
			}

			proto := map[string]int{"i": i, "j": j}
			emplaceRoom(dungeon, proto)
		}
	}
}

func emplaceRoom(dungeon *map[string]any, proto map[string]int) {
	if (*dungeon)["n_rooms"].(int) == 999 {
		return
	}

	var r int
	var c int

	setRoom(dungeon, &proto)
	r1 := (proto["i"] * 2) + 1
	c1 := (proto["j"] * 2) + 1
	r2 := ((proto["i"] + proto["height"]) * 2) - 1
	c2 := ((proto["j"] + proto["width"]) * 2) - 1

	if (r1 < 1) || (r2 > (*dungeon)["max_row"].(int)) {
		return
	}

	if (c1 < 1) || (c2 > (*dungeon)["max_col"].(int)) {
		return
	}

	hit := soundRoom(dungeon, r1, c1, r2, c2)

	_, ok := hit["blocked"]

	if ok {
		return
	}

	hitList := KeysFromMap(hit)
	nHits := len(hitList)
	var roomId int

	if nHits == 0 {
		roomId = (*dungeon)["n_rooms"].(int) + 1
		(*dungeon)["n_rooms"] = roomId
	} else {
		return
	}

	(*dungeon)["last_room_id"] = roomId

	for r := r1; r <= r2; r++ {
		for c := c1; c <= c2; c++ {
			if ((*dungeon)["cell"].([][]any)[r][c].(int) & ENTRANCE) != 0 {
				(*dungeon)["cell"].([][]any)[r][c] = (*dungeon)["cell"].([][]any)[r][c].(int) & ^ESPACE
			} else if ((*dungeon)["cell"].([][]any)[r][c].(int) & PERIMETER) != 0 {
				(*dungeon)["cell"].([][]any)[r][c] = (*dungeon)["cell"].([][]any)[r][c].(int) & ^PERIMETER
			}

			(*dungeon)["cell"].([][]any)[r][c] = (*dungeon)["cell"].([][]any)[r][c].(int) | ROOM | (roomId << 6)
		}
	}

	height := ((r2 - r1) + 1) * 10
	width := ((c2 - c1) + 1) * 10

	roomData := map[string]int{
		"id": roomId, "row": r1, "col": c1,
		"north": r1, "south": r2, "west": c1, "east": c2,
		"height": height, "width": width, "area": (height * width),
	}

	if (*dungeon)["room"] == nil {
		(*dungeon)["room"] = map[int]map[string]int{}
	}

	(*dungeon)["room"].(map[int]map[string]int)[roomId] = roomData

	for r := r1 - 1; r <= r2+1; r++ {
		if ((*dungeon)["cell"].([][]any)[r][c1-1].(int) & (ROOM | ENTRANCE)) == 0 {
			(*dungeon)["cell"].([][]any)[r][c1-1] = (*dungeon)["cell"].([][]any)[r][c1-1].(int) | PERIMETER
		}

		if ((*dungeon)["cell"].([][]any)[r][c2+1].(int) & (ROOM | ENTRANCE)) == 0 {
			(*dungeon)["cell"].([][]any)[r][c2+1] = (*dungeon)["cell"].([][]any)[r][c2+1].(int) | PERIMETER
		}
	}

	for c := c1 - 1; c <= c2+1; c++ {
		if ((*dungeon)["cell"].([][]any)[r1-1][c].(int) & (ROOM | ENTRANCE)) == 0 {
			(*dungeon)["cell"].([][]any)[r1-1][c] = (*dungeon)["cell"].([][]any)[r1-1][c].(int) | PERIMETER
		}

		if ((*dungeon)["cell"].([][]any)[r2+1][c].(int) & (ROOM | ENTRANCE)) == 0 {
			(*dungeon)["cell"].([][]any)[r2+1][c] = (*dungeon)["cell"].([][]any)[r2+1][c].(int) | PERIMETER
		}
	}
}

func soundRoom(dungeon *map[string]any, r1, c1, r2, c2 int) map[string]int {
	hit := make(map[string]int)

	for r := r1; r <= r2; r++ {
		for c := c1; c <= c2; c++ {
			if ((*dungeon)["cell"].([][]any)[r][c].(int) & BLOCKED) != 0 {
				return map[string]int{"blocked": 1}
			}

			if ((*dungeon)["cell"].([][]any)[r][c].(int) & ROOM) != 0 {
				id := ((*dungeon)["cell"].([][]any)[r][c].(int) & ROOM_ID) >> 6
				_, ok := hit[string(id)]

				if !ok {
					hit[string(id)] = 0
				}

				hit[string(id)] += 1
			}
		}
	}

	return hit
}

func setRoom(dungeon *map[string]any, proto *map[string]int) {
	base := (*dungeon)["room_base"].(int)
	radix := (*dungeon)["room_radix"].(int)

	_, ok := (*proto)["height"]

	if !ok {
		i, ok := (*proto)["i"]

		if ok {
			a := (*dungeon)["n_i"].(int) - base - i

			if a < 0 {
				a = 0
			}

			var r int

			if a < radix {
				r = a
			} else {
				r = radix
			}

			(*proto)["height"] = (*(*dungeon)["rand"].(*rand.Rand)).Intn(r) + base
		} else {
			(*proto)["height"] = (*(*dungeon)["rand"].(*rand.Rand)).Intn(radix) + base
		}
	}

	_, ok = (*proto)["width"]

	if !ok {
		j, ok := (*proto)["j"]

		if ok {
			a := (*dungeon)["n_j"].(int) - base - j

			if a < 0 {
				a = 0
			}

			var r int

			if a < radix {
				r = a
			} else {
				r = radix
			}

			(*proto)["width"] = (*(*dungeon)["rand"].(*rand.Rand)).Intn(r) + base
		} else {
			(*proto)["width"] = (*(*dungeon)["rand"].(*rand.Rand)).Intn(radix) + base
		}
	}

	_, ok = (*proto)["i"]

	if !ok {
		(*proto)["i"] = (*(*dungeon)["rand"].(*rand.Rand)).Intn((*dungeon)["n_i"].(int) - (*proto)["height"])
	}

	_, ok = (*proto)["j"]

	if !ok {
		(*proto)["j"] = (*(*dungeon)["rand"].(*rand.Rand)).Intn((*dungeon)["n_j"].(int) - (*proto)["width"])
	}
}

func scatterRooms(dungeon *map[string]any) {
	n_rooms := allocRooms(dungeon)

	for i := 0; i < n_rooms; i++ {
		emplaceRoom(dungeon, nil)
	}
}

func allocRooms(dungeon *map[string]any) int {
	dungeonArea := (*dungeon)["n_cols"].(int) * (*dungeon)["n_rows"].(int)
	roomArea := (*dungeon)["room_max"].(int) * (*dungeon)["room_max"].(int)
	return int(dungeonArea / roomArea)
}

func openRooms(dungeon *map[string]any) {
	for id := 1; id <= (*dungeon)["n_rooms"].(int); id++ {
		openRoom(dungeon, (*dungeon)["room"].(map[int]map[string]int)[id])
	}

	delete(*dungeon, "connect")
}

func openRoom(dungeon *map[string]any, room map[string]int) {
	list := doorSills(dungeon, room)

	if len(list) == 0 {
		return
	}

	n_opens = allocOpens(dungeon, room)

	for i := 0; i < n_opens; i++ {
		sill := list[(*(*dungeon)["rand"].(*rand.Rand)).Intn(len(list))]

		if len(sill) == 0 {
			break
		}

		door_r := sill["door_r"]
		door_c := sill["door_c"]
		door_cell := (*dungeon)["cell"].([][]any)[door_r][door_c].(int)

		if (door_cell & DOORSPACE) != 0 {
			i -= 1
			continue
		}

		if out_id := sill["out_id"]; out_id != 0 {
		}
	}
}

func doorSills(dungeon *map[string]any, room map[string]int) []map[string]int {

}

func allocOpens(dungeon *map[string]any, room map[string]int) int {

}

func labelRooms(dungeon *map[string]any) {

}

func corridors(dungeon *map[string]any) {

}

func emplaceStairs(dungeon *map[string]any) {

}

func cleanDungeon(dungeon *map[string]any) {

}

func imageDungeon(dungeon map[string]any) {

}
