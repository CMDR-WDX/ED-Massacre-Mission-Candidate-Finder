package evaluation

import "massacre-finder/dataBuilder"
import "massacre-finder/args"

type SystemEvaluationResult struct {
	Score                          float32                   `json:"score,omitempty"`
	AnarchyFactionName             string                    `json:"anarchyFactionName,omitempty"`
	SystemName                     string                    `json:"systemName,omitempty"`
	SourceSystemAnarchyFactionCount int 					 `json:"sourceSystemAnarchyFactionCount"`
	SourcingFactionsCount          int                       `json:"sourcingFactionsCount,omitempty"`
	SourcingSystems                int                       `json:"sourcingSystems,omitempty"`
	ExternalSystemCount            int                       `json:"externalSystemCount,omitempty"`
	ExternalSystemCountWithAnarchy int                       `json:"externalSystemCountWithAnarchy,omitempty"`
	Rings                          int                       `json:"rings,omitempty"`
	MetaSurroundingSystems         []dataBuilder.EliteSystem `json:"metaSurroundingSystems,omitempty"`
	MetaSystem                     dataBuilder.EliteSystem   `json:"metaSystem"`
}

// EvaluateSystem evaluates the current Systems "goodness" for being a Stacking System.
func EvaluateSystem(system dataBuilder.EliteSystem, dataStore map[dataBuilder.EliteSector][]dataBuilder.EliteSystem, config args.Args) (SystemEvaluationResult, bool) {

	// Do a check to see if this System is a good dest. candidate.
	if system.AnarchyFactionCount != 1 {
		return SystemEvaluationResult{}, false // Not a valid candidate
	}

	if config.FilterOnlyRingedSource && system.RingQty == 0 {
		return SystemEvaluationResult{}, false // No Rings
	}

	score := float32(0)

	if system.RingQty > 0 {
		score += 2 - 1.0/float32(system.RingQty)
	}

	// Contains all Systems around the target system within a 10ly radius
	populatedSystemsInRange := getAllPopulatedSystemsIn10LyRadius(system, dataStore)

	if len(populatedSystemsInRange) < config.MinSourceSystemCount {
		return SystemEvaluationResult{}, false
	}



	stationCount := 0
	sourceSystemAnarchyCount := 0
	for _, sys := range populatedSystemsInRange {
		sourceSystemAnarchyCount += int(sys.AnarchyFactionCount)
		for _, station := range sys.Stations {
			if  station.Distance < float32(config.MaxDistanceInLsForStationToBeConsidered) {
				stationCount++
			}
		}
	}

	if stationCount < config.MinSourceStationCount {
		return SystemEvaluationResult{}, false
	}

	var systemToSurroundingSystemsLookup = make(map[uint64]dataBuilder.EliteSystem)

	// Go through all Systems that are accessible by the Source systems
	for _, newSystem := range populatedSystemsInRange {
		systemsOfGivenSystemInRange := getAllPopulatedSystemsIn10LyRadius(newSystem, dataStore)
		for _, s := range systemsOfGivenSystemInRange {
			if sysDistanceSquared(s, system) > 10*10 {
				systemToSurroundingSystemsLookup[s.Id] = s
			}
		}
	}
	// Find all the Systems that are destination systems but not source systems -> > 10ly from current system
	outsideSystemCount := 0
	outsideSystemAnarchyCount := 0

	//////////////////// Negative Calculations ///////////////////////////

	for _, outsideSystem := range systemToSurroundingSystemsLookup {
		outsideSystemCount++
		outsideSystemAnarchyCount += int(outsideSystem.AnarchyFactionCount)
	}

	if outsideSystemCount > config.MaxOtherDestSystemsForSource {
		return SystemEvaluationResult{}, false
	}

	if outsideSystemAnarchyCount > config.MaxOtherDestSystemsForSourceAnarchyCount {
		return SystemEvaluationResult{}, false
	}

	score -= float32(outsideSystemAnarchyCount*outsideSystemAnarchyCount) + float32(outsideSystemCount)

	// Find the "inside" anarchy count
	insideSystemAnarchyCount := 0
	for _, s := range populatedSystemsInRange {
		insideSystemAnarchyCount += int(s.AnarchyFactionCount)
	}

	score -= float32(insideSystemAnarchyCount)

	//////////////////// Positive Calculations ///////////////////////////
	// inverse mapping of faction count and qty to score
	nonAnarchyFactionQtyMapping := make(map[string]int)
	for _, s := range populatedSystemsInRange {

		for _, f := range s.NonAnarchyFactionNames {
			_, exists := nonAnarchyFactionQtyMapping[f]
			if !exists {
				nonAnarchyFactionQtyMapping[f] = 0
			}

			nonAnarchyFactionQtyMapping[f]++
		}
	}

	for _, count := range nonAnarchyFactionQtyMapping {
		score += 2 - (1.0 / float32(count))
	}

	// Do a pre-check to see if it's even worth to do further analysis on this system.
	return SystemEvaluationResult{
		AnarchyFactionName:             system.AnarchyFactionNames[0],
		SystemName:                     system.Name,
		SourcingFactionsCount:          len(nonAnarchyFactionQtyMapping),
		SourceSystemAnarchyFactionCount: sourceSystemAnarchyCount,
		ExternalSystemCount:            outsideSystemCount,
		ExternalSystemCountWithAnarchy: outsideSystemAnarchyCount,
		Rings:                          int(system.RingQty),
		SourcingSystems:                len(populatedSystemsInRange),
		Score:                          score,
		MetaSurroundingSystems:         populatedSystemsInRange,
		MetaSystem:                     system,
	}, true
}

func getAllPopulatedSystemsIn10LyRadius(system dataBuilder.EliteSystem, dataStore map[dataBuilder.EliteSector][]dataBuilder.EliteSystem) []dataBuilder.EliteSystem {
	returnSystems := make([]dataBuilder.EliteSystem, 0)

	// Get the Systems Sector and all surrounding Sectors
	ownSector := dataBuilder.BuildSector(system)
	sectors := buildSectorCube(ownSector)

	for _, sector := range sectors {
		listOfSystemsInSector, hasKey := dataStore[sector]

		if hasKey {
			returnSystems = appendSystems(system, 10, listOfSystemsInSector, returnSystems)
		}
	}

	return returnSystems
}

func appendSystems(systemToIgnore dataBuilder.EliteSystem, maxDistance float32, newSystems []dataBuilder.EliteSystem, oldSystems []dataBuilder.EliteSystem) []dataBuilder.EliteSystem {
	returnArray := oldSystems
	maxDistanceSquared := maxDistance * maxDistance

	for _, system := range newSystems {
		if system.Id == systemToIgnore.Id {
			continue
		}
		if sysDistanceSquared(system, systemToIgnore) > maxDistanceSquared {
			continue
		}

		returnArray = append(returnArray, system)
	}

	return returnArray
}

func sysDistanceSquared(a dataBuilder.EliteSystem, b dataBuilder.EliteSystem) float32 {
	x := a.X - b.X
	y := a.Y - b.Y
	z := a.Z - b.Z

	return x*x + y*y + z*z
}

func buildSectorCube(sector dataBuilder.EliteSector) []dataBuilder.EliteSector {
	return []dataBuilder.EliteSector{
		// Z = -1
		*dataBuilder.NewEliteSector(sector.X-1, sector.Y-1, sector.Z-1),
		*dataBuilder.NewEliteSector(sector.X-0, sector.Y-1, sector.Z-1),
		*dataBuilder.NewEliteSector(sector.X+1, sector.Y-1, sector.Z-1),

		*dataBuilder.NewEliteSector(sector.X-1, sector.Y-0, sector.Z-1),
		*dataBuilder.NewEliteSector(sector.X-0, sector.Y-0, sector.Z-1),
		*dataBuilder.NewEliteSector(sector.X+1, sector.Y-0, sector.Z-1),

		*dataBuilder.NewEliteSector(sector.X-1, sector.Y+1, sector.Z-1),
		*dataBuilder.NewEliteSector(sector.X-0, sector.Y+1, sector.Z-1),
		*dataBuilder.NewEliteSector(sector.X+1, sector.Y+1, sector.Z-1),
		// Z = 0
		*dataBuilder.NewEliteSector(sector.X-1, sector.Y-1, sector.Z),
		*dataBuilder.NewEliteSector(sector.X-0, sector.Y-1, sector.Z),
		*dataBuilder.NewEliteSector(sector.X+1, sector.Y-1, sector.Z),

		*dataBuilder.NewEliteSector(sector.X-1, sector.Y-0, sector.Z),
		sector,
		*dataBuilder.NewEliteSector(sector.X+1, sector.Y-0, sector.Z),

		*dataBuilder.NewEliteSector(sector.X-1, sector.Y+1, sector.Z),
		*dataBuilder.NewEliteSector(sector.X-0, sector.Y+1, sector.Z),
		*dataBuilder.NewEliteSector(sector.X+1, sector.Y+1, sector.Z),

		// Z = 1
		*dataBuilder.NewEliteSector(sector.X-1, sector.Y-1, sector.Z+1),
		*dataBuilder.NewEliteSector(sector.X-0, sector.Y-1, sector.Z+1),
		*dataBuilder.NewEliteSector(sector.X+1, sector.Y-1, sector.Z+1),

		*dataBuilder.NewEliteSector(sector.X-1, sector.Y-0, sector.Z+1),
		*dataBuilder.NewEliteSector(sector.X-0, sector.Y-0, sector.Z+1),
		*dataBuilder.NewEliteSector(sector.X+1, sector.Y-0, sector.Z+1),

		*dataBuilder.NewEliteSector(sector.X-1, sector.Y+1, sector.Z+1),
		*dataBuilder.NewEliteSector(sector.X-0, sector.Y+1, sector.Z+1),
		*dataBuilder.NewEliteSector(sector.X+1, sector.Y+1, sector.Z+1),
	}

}
