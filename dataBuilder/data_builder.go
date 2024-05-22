package dataBuilder

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"massacre-finder/args"
	"math"
	"os"
	"sort"
	"strconv"
)

type EliteSystemStation struct {
	Distance       float32 `json:"distance"`
	Type           string  `json:"type"`
	PrimaryEconomy string  `json:"primaryEconomy"`
}

type EliteSystem struct {
	Id                     uint64               `json:"id,omitempty"`
	Name                   string               `json:"name,omitempty"`
	X                      float32              `json:"x,omitempty"`
	Y                      float32              `json:"y,omitempty"`
	Z                      float32              `json:"z,omitempty"`
	AnarchyFactionCount    int8                 `json:"anarchyFactionCount,omitempty"`
	NonAnarchyFactionCount int8                 `json:"nonAnarchyFactionCount,omitempty"`
	RingQty                int8                 `json:"ringQty,omitempty"`
	SystemSecurityLevel    int8                 `json:"systemSecurityLevel,omitempty"`
	AnarchyFactionNames    []string             `json:"anarchyFactionNames,omitempty"`
	NonAnarchyFactionNames []string             `json:"nonAnarchyFactionNames,omitempty"`
	Stations               []EliteSystemStation `json:"stations"`
}
type EliteSector struct {
	X int `json:"X"`
	Y int `json:"Y"`
	Z int `json:"Z"`
}

func NewEliteSector(x int, y int, z int) *EliteSector {
	return &EliteSector{X: x, Y: y, Z: z}
}

type eliteSystemJSONCoords struct {
	X float32 `json:"X"`
	Y float32 `json:"Y"`
	Z float32 `json:"Z"`
}

type eliteSystemJSONFaction struct {
	Name       string `json:"Name"`
	Government string `json:"government"`
}

type eliteSystemJSONBodyRingEntry struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type eliteSystemStationEntry struct {
	DistanceToArrival float32 `json:"distanceToArrival"`
	Type              string  `json:"type"`
	PrimaryEconomy	  string  `json:"primaryEconomy"`
	Services		 []string `json:"services"`
}

type eliteSystemJSONBody struct {
	Name  string                         `json:"Name"`
	Type  string                         `json:"type"`
	Rings []eliteSystemJSONBodyRingEntry `json:"rings"`
	Stations []eliteSystemStationEntry `json:"stations"`
}

type EliteSystemJSON struct {
	Coords   eliteSystemJSONCoords     `json:"coords"`
	Name     string                    `json:"Name"`
	Security string                    `json:"security"`
	Factions []eliteSystemJSONFaction  `json:"factions"`
	Bodies   []eliteSystemJSONBody     `json:"bodies"`
	Id       uint64                    `json:"id64"`
	Stations []eliteSystemStationEntry `json:"stations"`
}

type eliteSystemJSONEntry struct {
	value EliteSystemJSON
	next  *eliteSystemJSONEntry
}

type eliteSystemJSONList struct {
	first *eliteSystemJSONEntry
	head  *eliteSystemJSONEntry
}

// BuildSystemData reads the populated JSON System by System (to reduce RAM usage) and store just the relevant data.
// Said relevant data will then be returned, namely a Map of Sector-Index to List of Systems in the Sector
// The Sector-Size is set to 10, which is convenient because the max distance for Massacre Source Systems is 10ly.
func buildSystemData(filepath string) []EliteSystemJSON {
	file, err := os.OpenFile(filepath, os.O_RDONLY, os.ModePerm)

	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	systemsParsedCounter := 0

	allSystems := eliteSystemJSONList{
		first: nil, head: nil,
	}

	const bufferSize = 30_000_000
	buffer := [bufferSize]rune{}
	nexCharPtr := 0
	currentDepth := 0 // Outer Array is not needed. That's why it's -1
	isInWrapperArray := false

	for {
		if char, _, err := reader.ReadRune(); err != nil {
			if err == io.EOF {
				break
			} else {
				log.Fatalln(err)
			}
		} else {
			if char == '[' && !isInWrapperArray {
				isInWrapperArray = true
				continue
			} else if char == ']' && currentDepth == 0 {
				// TODO: Close down
			}

			if !isInWrapperArray {
				continue
			}

			if char == '{' || char == '[' {
				currentDepth++
			} else if char == '}' || char == ']' {
				currentDepth--
			}

			buildString := false

			if currentDepth == 0 {
				if nexCharPtr == 0 {
					continue
				}
				// There is Data present. Append the "closing" char and turn into a string
				buildString = true
			}

			buffer[nexCharPtr] = char
			nexCharPtr++

			if buildString {
				stringArray := make([]rune, nexCharPtr)
				for i := 0; i < nexCharPtr; i++ {
					stringArray[i] = buffer[i]
				}
				nexCharPtr = 0
				asString := string(stringArray)

				var jsonData EliteSystemJSON
				err := json.Unmarshal([]byte(asString), &jsonData)
				if err != nil {
					print(err)
				} else {

					systemsParsedCounter++

					entry := eliteSystemJSONEntry{
						value: jsonData,
						next:  nil,
					}

					if allSystems.first == nil {
						allSystems.first = &entry
						allSystems.head = &entry
					} else {
						allSystems.head.next = &entry
						allSystems.head = &entry
					}
				}
			}

		}
	}

	println("Parsed a total of " + strconv.Itoa(systemsParsedCounter) + " Systems. Copying over...")

	// Build Array from Linked List
	returnArray := make([]EliteSystemJSON, systemsParsedCounter)

	index := 0
	pointer := allSystems.first
	for {
		if pointer == nil {
			break
		}
		returnArray[index] = pointer.value
		index++
		pointer = pointer.next
	}

	return returnArray
}

func buildCacheFile(cacheFile string, sourceFile string) {
	if _, err := os.Stat(cacheFile); err == nil {
		// File Exist, Delete
		err := os.Remove(cacheFile)
		if err != nil {
			fmt.Println(err)
		}
	}
	newData := buildSystemData(sourceFile)

	jsonString, _ := json.Marshal(newData)
	ioutil.WriteFile(cacheFile, jsonString, os.ModePerm)
}

func GetOrCreateSystemData(cachePath string, sourcePath string, forceRebuild bool) []EliteSystemJSON {
	_, err := os.Stat(cachePath)
	doesFileExist := err == nil

	isRebuildNeeded := !doesFileExist || forceRebuild

	if isRebuildNeeded {
		buildCacheFile(cachePath, sourcePath)
	}

	// and now get the cache

	var data []EliteSystemJSON

	file, _ := ioutil.ReadFile(cachePath)

	json.Unmarshal(file, &data)

	return data
}

func BuildSectoredData(systemsAsList []EliteSystemJSON, config args.Args) map[EliteSector][]EliteSystem {

	var returnMap = make(map[EliteSector][]EliteSystem)

	for _, entry := range systemsAsList {
		sector := buildSector(entry.Coords.X, entry.Coords.Y, entry.Coords.Z)
		listOfSector, hasList := returnMap[sector]

		if hasList {
			returnMap[sector] = append(listOfSector, buildSystem(entry, config))
		} else {
			returnMap[sector] = []EliteSystem{buildSystem(entry, config)}
		}
	}

	return returnMap
}

func buildSector(x float32, y float32, z float32) EliteSector {
	const SectorSize = 10
	_x := int(math.Floor(float64(x / SectorSize)))
	_y := int(math.Floor(float64(y / SectorSize)))
	_z := int(math.Floor(float64(z / SectorSize)))
	return EliteSector{
		X: _x,
		Y: _y,
		Z: _z,
	}
}

func BuildSector(system EliteSystem) EliteSector {
	return buildSector(system.X, system.Y, system.Z)
}

func buildSystem(data EliteSystemJSON, config args.Args) EliteSystem {
	returnVal := EliteSystem{}
	returnVal.X = data.Coords.X
	returnVal.Y = data.Coords.Y
	returnVal.Z = data.Coords.Z
	returnVal.Name = data.Name
	returnVal.Id = data.Id

	// Calculate Faction Count
	returnVal.AnarchyFactionCount = 0
	returnVal.NonAnarchyFactionCount = 0
	returnVal.NonAnarchyFactionNames = make([]string, 0)
	returnVal.AnarchyFactionNames = make([]string, 0)

	jsonStations := make([]eliteSystemStationEntry, 0)
	jsonStations = append(jsonStations, data.Stations...)

	for _, body := range data.Bodies {
		for _, s := range body.Stations {
			jsonStations = append(jsonStations, s)
		}
	}

	stations := make([]EliteSystemStation, 0)
	for _, st := range data.Stations {
		hasMissionBoard := false
		for _, service := range st.Services {
			if service == "Missions" {
				hasMissionBoard = true
				break
			}
		}
		if !hasMissionBoard {
			continue
		}
		if st.DistanceToArrival > float32(config.MaxDistanceInLsForStationToBeConsidered) {
			continue
		}
		/// Station Filter
		relevantStationTypes := []string{"Outpost", "Coriolis Starport", "Orbis Starport", "Ocellus Starport"}

		if config.ConsiderGroundBases {
			relevantStationTypes = append(relevantStationTypes, "Planetary Outpost")
		}

		if config.ConsiderOdysseySettlements {
			relevantStationTypes = append(relevantStationTypes, "Settlement")
		}

		isRelevantType := false

		for _, t := range relevantStationTypes {
			if st.Type == t {
				isRelevantType = true
				break
			}
		}

		if !isRelevantType {
			continue
		}



		newStation := EliteSystemStation{
			Distance:       st.DistanceToArrival,
			Type:           st.Type,
			PrimaryEconomy: st.PrimaryEconomy,
		}

		stations = append(stations, newStation)
	}

	sort.Slice(stations, func(i, j int) bool {
		return stations[i].Distance < stations[j].Distance
	})
	returnVal.Stations = stations

	for _, faction := range data.Factions {
		if faction.Government == "Anarchy" {
			returnVal.AnarchyFactionCount++
			returnVal.AnarchyFactionNames = append(returnVal.AnarchyFactionNames, faction.Name)

		} else {
			returnVal.NonAnarchyFactionCount++
			returnVal.NonAnarchyFactionNames = append(returnVal.NonAnarchyFactionNames, faction.Name)
		}
	}

	switch data.Security {
	case "Anarchy":
		returnVal.SystemSecurityLevel = 0
		break
	case "Low":
		returnVal.SystemSecurityLevel = 1
		break
	case "Medium":
		returnVal.SystemSecurityLevel = 2
		break
	case "High":
		returnVal.SystemSecurityLevel = 3
		break
	default:
		returnVal.SystemSecurityLevel = 127
	}

	returnVal.RingQty = 0
	for _, entry := range data.Bodies {
		if len(entry.Rings) > 0 {
			returnVal.RingQty++
		}
	}

	return returnVal
}
