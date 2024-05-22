package args

type Args struct {
	FilterOnlyRingedSource                   bool
	MinSourceSystemCount                     int
	MaxOtherDestSystemsForSource             int
	MaxOtherDestSystemsForSourceAnarchyCount int
	MinSourceStationCount                    int
	MaxDistanceInLsForStationToBeConsidered  int
	ConsiderGroundBases						 bool
	ConsiderOdysseySettlements				 bool
}
