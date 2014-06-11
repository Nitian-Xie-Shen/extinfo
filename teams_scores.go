package extinfo

import (
	"errors"
)

// TeamScore (team score) contains the name of the team and the score, i.e. flags scored in flag modes / points gained for holding bases in capture modes / frags achieved in DM modes / skulls collected
type TeamScore struct {
	Name  string // name of the team, e.g. "good"
	Score int    // amount of points (flags in ctf modes, frags in deathmatch modes, points in capture, skulls in collect)
	Bases []int  // the numbers/IDs of the bases the team possesses (only used in capture modes)
}

// TeamsScoresRaw (teams's scores) contains the game mode as raw int, the seconds left in the game, and a slice of TeamScores
type TeamsScoresRaw struct {
	GameMode int         // current game mode
	SecsLeft int         // the time left until intermission in seconds
	Scores   []TeamScore // a team score for each team
}

// TeamsScores (teams's scores) contains the game mode as human readable string, the seconds left in the game, and a slice of TeamScores
type TeamsScores struct {
	TeamsScoresRaw
	GameMode string // current game mode
}

// GetTeamsScoresRaw queries a Sauerbraten server at addr on port for the teams' names and scores and returns the raw response and/or an error in case something went wrong or the server is not running a team mode.
func (s *Server) GetTeamsScoresRaw() (teamsScoresRaw TeamsScoresRaw, err error) {
	teamsScoresRaw = TeamsScoresRaw{}

	request := buildRequest(EXTENDED_INFO, EXTENDED_INFO_TEAMS_SCORES, 0)
	response, err := s.queryServer(request)
	if err != nil {
		return
	}

	// ignore first 3 bytes: EXTENDED_INFO, EXTENDED_INFO_TEAMS_SCORES, EXTENDED_INFO_ACK
	response = response[3:]

	positionInResponse = 0

	// check for correct extinfo protocol version
	if dumpInt(response) != EXTENDED_INFO_VERSION {
		err = errors.New("extinfo: wrong extinfo protocol version")
		return
	}

	// next int describes wether the server runs a team mode or not
	isTeamMode := true
	if dumpInt(response) != 0 {
		isTeamMode = false
	}

	teamsScoresRaw.GameMode = dumpInt(response)
	teamsScoresRaw.SecsLeft = dumpInt(response)

	if !isTeamMode {
		// no team scores following
		err = errors.New("extinfo: server is not running a team mode")
		return
	}

	for response[positionInResponse] != 0x0 {
		name := dumpString(response)
		score := dumpInt(response)
		numBases := dumpInt(response)

		bases := make([]int, 0)

		for i := 0; i < numBases; i++ {
			bases = append(bases, dumpInt(response))
		}

		teamsScoresRaw.Scores = append(teamsScoresRaw.Scores, TeamScore{name, score, bases})
	}

	return
}

// GetTeamsScores queries a Sauerbraten server at addr on port for the teams' names and scores and returns the parsed response and/or an error in case something went wrong or the server is not running a team mode. Parsed response means that the int value sent as game mode is translated into the human readable name, e.g. '12' -> "insta ctf".
func (s *Server) GetTeamsScores() (TeamsScores, error) {
	teamsScores := TeamsScores{}

	teamsScoresRaw, err := s.GetTeamsScoresRaw()
	if err != nil {
		return teamsScores, err
	}

	teamsScores.TeamsScoresRaw = teamsScoresRaw
	teamsScores.GameMode = getGameModeName(teamsScoresRaw.GameMode)

	return teamsScores, nil
}
