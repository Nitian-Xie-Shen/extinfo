// Package extinfo provides easy access to the state information of a Sauerbraten game server (called 'extinfo' in the Sauerbraten source code).
package extinfo

import (
	"errors"
	"net"
)

// the current position in a response ([]byte)
// needed, since values are encoded in variable amount of bytes
// global to not have to pass around an int on every dump
var positionInResponse int

// Constants describing the type of information to query for
const (
	EXTENDED_INFORMATION = 0
	BASIC_INFORMATION = 1
)

// Constants describing the type of extended information to query for
const (
	UPTIME = 0
	PLAYERSTATS = 1
	TEAMSCORE = 2
)

// GetTeamsScores queries a Sauerbraten server at addr on port for the teams' names and scores and returns the parsed response and/or an error in case something went wrong or the server is not running a team mode. Parsed response means that the int value sent as game mode is translated into the human readable name, e.g. '12' -> "insta ctf".
func GetTeamsScores(addr string, port int) (TeamsScores, error) {
	teamsScoresRaw, err := GetTeamsScoresRaw(addr, port)
	teamsScores := TeamsScores{getGameModeName(teamsScoresRaw.GameMode), teamsScoresRaw.SecsLeft, teamsScoresRaw.Scores}
	return teamsScores, err
}

// GetTeamsScoresRaw queries a Sauerbraten server at addr on port for the teams' names and scores and returns the raw response and/or an error in case something went wrong or the server is not running a team mode.
func GetTeamsScoresRaw(addr string, port int) (TeamsScoresRaw, error) {
	teamsScoresRaw := TeamsScoresRaw{}

	response, err := queryServer(addr, port, buildRequest(EXTENDED_INFORMATION, 2, 0))
	if err != nil {
		return teamsScoresRaw, err
	}

	positionInResponse = 0

	// first int is EXTENDED_INFORMATION = 0
	_ = dumpInt(response)

	// next int is TEAMSCORE = 2
	_ = dumpInt(response)

	// next int is EXT_ACK = -1
	_ = dumpInt(response)

	// next int is EXT_VERSION
	_ = dumpInt(response)

	// next int describes wether the server runs a team mode or not
	isTeamMode := true
	if dumpInt(response) != 0 {
		isTeamMode = false
	}

	teamsScoresRaw.GameMode = dumpInt(response)
	teamsScoresRaw.SecsLeft = dumpInt(response)

	if !isTeamMode {
		// no team scores following
		return teamsScoresRaw, errors.New("server is not running a team mode")
	}

	name := ""
	score := 0

	for positionInResponse < len(response) {
		name = dumpString(response)
		score = dumpInt(response)

		bases := make([]int, 0)

		for i := 0; i < dumpInt(response); i++ {
			bases = append(bases, dumpInt(response))
		}

		teamsScoresRaw.Scores = append(teamsScoresRaw.Scores, TeamScore{name, score, bases})
	}

	return teamsScoresRaw, nil
}


// GetBasicInfo queries a Sauerbraten server at addr on port and returns the parsed response or an error in case something went wrong. Parsed response means that the int values sent as game mode and master mode are translated into the human readable name, e.g. '12' -> "insta ctf".
func GetBasicInfo(addr string, port int) (BasicInfo, error) {
	basicInfo := BasicInfo{}

	response, err := queryServer(addr, port, buildRequest(BASIC_INFORMATION, 0, 0))
	if err != nil {
		return basicInfo, err
	}

	positionInResponse = 0

	// first int is BASIC_INFORMATION = 1
	_ = dumpInt(response)

	basicInfo.NumberOfClients = dumpInt(response)
	// next int is always 5, the number of additional attributes after the playercount and the strings for map and description
	//numberOfAttributes := dumpInt(response)
	_ = dumpInt(response)
	basicInfo.ProtocolVersion = dumpInt(response)
	basicInfo.GameMode = getGameModeName(dumpInt(response))
	basicInfo.SecsLeft = dumpInt(response)
	basicInfo.MaxNumberOfClients = dumpInt(response)
	basicInfo.MasterMode = getMasterModeName(dumpInt(response))
	basicInfo.Map = dumpString(response)
	basicInfo.Description = dumpString(response)

	return basicInfo, nil
}

// GetBasicInfoRaw queries a Sauerbraten server at addr on port and returns the raw response or an error in case something went wrong. Raw response means that the int values sent as game mode and master mode are NOT translated into the human readable name.
func GetBasicInfoRaw(addr string, port int) (BasicInfoRaw, error) {
	basicInfoRaw := BasicInfoRaw{}

	response, err := queryServer(addr, port, buildRequest(BASIC_INFORMATION, 0, 0))
	if err != nil {
		return basicInfoRaw, err
	}

	positionInResponse = 0

	// first int is BASIC_INFORMATION = 1
	_ = dumpInt(response)
	basicInfoRaw.NumberOfClients = dumpInt(response)
	// next int is always 5, the number of additional attributes after the playercount and the strings for map and description
	//numberOfAttributes := dumpInt(response)
	_ = dumpInt(response)
	basicInfoRaw.ProtocolVersion = dumpInt(response)
	basicInfoRaw.GameMode = dumpInt(response)
	basicInfoRaw.SecsLeft = dumpInt(response)
	basicInfoRaw.MaxNumberOfClients = dumpInt(response)
	basicInfoRaw.MasterMode = dumpInt(response)
	basicInfoRaw.Map = dumpString(response)
	basicInfoRaw.Description = dumpString(response)

	return basicInfoRaw, nil
}

// GetUptime returns the uptime of the server in seconds.
func GetUptime(addr string, port int) (int, error) {
	response, err := queryServer(addr, port, buildRequest(EXTENDED_INFORMATION, UPTIME, 0))
	if err != nil {
		return -1, err
	}

	positionInResponse = 0

	// first int is EXTENDED_INFORMATION
	_ = dumpInt(response)

	// next int is EXT_UPTIME = 0
	_ = dumpInt(response)

	// next int is EXT_ACK = -1
	_ = dumpInt(response)

	// next int is EXT_VERSION
	_ = dumpInt(response)

	// next int is the actual uptime
	uptime := dumpInt(response)

	return uptime, nil
}

// GetPlayerInfo returns the parsed information about the player with the given clientNum.
func GetPlayerInfo(addr string, port int, clientNum int) (PlayerInfo, error) {
	playerInfo := PlayerInfo{}

	response, err := queryServer(addr, port, buildRequest(EXTENDED_INFORMATION, PLAYERSTATS, clientNum))
	if err != nil {
		return playerInfo, err
	}

	if response[5] != 0x00 {
		// there was an error
		return playerInfo, errors.New("invalid cn")
	}

	// throw away 7 first ints (EXTENDED_INFORMATION, PLAYERSTATS, clientNum, server ACK byte, server VERSION byte, server NO_ERROR byte, server PLAYERSTATS_RESP_STATS byte)
	response = response[7:]

	playerInfo = parsePlayerInfo(response)

	return playerInfo, nil
}

// GetPlayerInfoRaw returns the raw information about the player with the given clientNum.
func GetPlayerInfoRaw(addr string, port int, clientNum int) (PlayerInfoRaw, error) {
	playerInfoRaw := PlayerInfoRaw{}

	response, err := queryServer(addr, port, buildRequest(EXTENDED_INFORMATION, PLAYERSTATS, clientNum))
	if err != nil {
		return playerInfoRaw, err
	}

	if response[5] != 0x00 {
		// there was an error
		return playerInfoRaw, errors.New("invalid cn")
	}

	// throw away 7 first ints (EXTENDED_INFORMATION, PLAYERSTATS, clientNum, server ACK byte, server VERSION byte, server NO_ERROR byte, server PLAYERSTATS_RESP_STATS byte)
	response = response[7:]
	
	positionInResponse = 0

	playerInfoRaw.ClientNum = dumpInt(response)
	playerInfoRaw.Ping = dumpInt(response)
	playerInfoRaw.Name = dumpString(response)
	playerInfoRaw.Team = dumpString(response)
	playerInfoRaw.Frags = dumpInt(response)
	playerInfoRaw.Flags = dumpInt(response)
	playerInfoRaw.Deaths = dumpInt(response)
	playerInfoRaw.Teamkills = dumpInt(response)
	playerInfoRaw.Damage = dumpInt(response)
	playerInfoRaw.Health = dumpInt(response)
	playerInfoRaw.Armour = dumpInt(response)
	playerInfoRaw.Weapon = dumpInt(response)
	playerInfoRaw.Privilege = dumpInt(response)
	playerInfoRaw.State = dumpInt(response)
	// IP from next 4 bytes
	ip := response[positionInResponse:positionInResponse+4]
	playerInfoRaw.IP = net.IPv4(ip[0], ip[1], ip[2], ip[3])

	return playerInfoRaw, nil
}

// own function, because it is used in GetPlayerInfo() + GetAllPlayerInfo()
func parsePlayerInfo(response []byte) PlayerInfo {
	playerInfo := PlayerInfo{}

	positionInResponse = 0

	playerInfo.ClientNum = dumpInt(response)
	playerInfo.Ping = dumpInt(response)
	playerInfo.Name = dumpString(response)
	playerInfo.Team = dumpString(response)
	playerInfo.Frags = dumpInt(response)
	playerInfo.Flags = dumpInt(response)
	playerInfo.Deaths = dumpInt(response)
	playerInfo.Teamkills = dumpInt(response)
	playerInfo.Damage = dumpInt(response)
	playerInfo.Health = dumpInt(response)
	playerInfo.Armour = dumpInt(response)
	playerInfo.Weapon = getWeaponName(dumpInt(response))
	playerInfo.Privilege = getPrivilegeName(dumpInt(response))
	playerInfo.State = getStateName(dumpInt(response))
	// IP from next 4 bytes
	ipBytes := response[positionInResponse:positionInResponse+4]
	playerInfo.IP = net.IPv4(ipBytes[0], ipBytes[1], ipBytes[2], ipBytes[3])

	return playerInfo
}

// GetAllPlayerInfo returns the Information of all Players (including spectators) as a []PlayerInfo
func GetAllPlayerInfo(addr string, port int) ([]PlayerInfo, error) {
	allPlayerInfo := []PlayerInfo{}

	response, err := queryServer(addr, port, buildRequest(EXTENDED_INFORMATION, PLAYERSTATS, -1))
	if err != nil {
		return allPlayerInfo, err
	}

	// response is multiple 64-byte responses, one for each player
	playerCount := len(response) / 64

	// parse each 64 byte packet (without the first 7 bytes) on its own and append to allPlayerInfo
	for i := 0; i < playerCount; i++ {
		allPlayerInfo = append(allPlayerInfo, parsePlayerInfo(response[i*64+7:(i*64)+64]))
	}

	return allPlayerInfo, nil
}
