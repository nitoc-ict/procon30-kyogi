package main

import (
	"fmt"
  "net"
  "bytes"
	"encoding/json"
	"io/ioutil"
	"log"
  "strconv"
  "net/http"
  "strings"
  "flag"
)

type FieldData struct {
	Width             int     `json:"width"`
	Height            int     `json:"height"`
	Points            [][]int `json:"points"`
	StartedAtUnixTime int     `json:"startedAtUnixTime"`
	Turn              int     `json:"turn"`
	Tiled             [][]int `json:"tiled"`
	Teams             []struct {
		TeamID int `json:"teamID"`
		Agents []struct {
			AgentID int `json:"agentID"`
			X       int `json:"x"`
			Y       int `json:"y"`
		} `json:"agents"`
		TilePoint int `json:"tilePoint"`
		AreaPoint int `json:"areaPoint"`
	} `json:"teams"`
	Actions []interface{} `json:"actions"`
}

type Action struct {
  AgentID int
  Type string
  Dx int
  Dy int
}

var myTeamID int
var maxTurn string

func main() {
  flag.Parse()
  args := flag.Args()

  var cntConect int;
  solverAnswer := make([]Action, 0)
  guiAnswer := make([]Action, 0)

  myTeamID, _ = strconv.Atoi(args[2])
  maxTurn = args[3]
  serverAddress := args[0] + ":" + args[1]

  fmt.Println(serverAddress)

  listener, err := net.Listen("tcp", serverAddress)

  if err != nil {
    fmt.Println("error")
    return
  }
  defer listener.Close()

  cntConect = 0

  for {
    connectClient(listener, args[4], &cntConect, &solverAnswer, &guiAnswer)
    go connectClient(listener, args[4], &cntConect, &solverAnswer, &guiAnswer)
  }
}

func connectClient(listener net.Listener, serverPORT string, cntConect *int, solverAnswer *[]Action, guiAnswer *[]Action) {
  conn, err := listener.Accept()


  if err != nil {
    fmt.Println("error")
    return
  }
  defer conn.Close()

  rsvData := make([]byte, 4018)
  n, err := conn.Read(rsvData)
  if err != nil {
    fmt.Println("error")
  }
  switch string(rsvData[0:2]) {
    case "sf" :
      conn.Write([]byte(convertJsonToSendData()))
    case "gf" :
      conn.Write([]byte(convertJsonToSendDataForGUI()))
    case "2s" :
      rsvSolverData(cntConect, string(rsvData[3:n]), serverPORT, solverAnswer, guiAnswer)
    case "2g" :
      rsvGUIData(cntConect, string(rsvData[3:n]), serverPORT, guiAnswer, solverAnswer)

  }
}

func rsvSolverData(cntConect *int, rsvData string, port string, answer *[]Action, otherAnswer *[]Action) {

  parseData := strings.Split(rsvData, ";")

  for i := 0; i < len(parseData) - 1; i++ {
    agentData := strings.Split(parseData[i], " ")

    agentID, _ := strconv.Atoi(agentData[0])
    moveType := agentData[1]
    dx, _ := strconv.Atoi(agentData[2])
    dy, _ := strconv.Atoi(agentData[3])

    *answer = append(*answer, Action{agentID, moveType, dx, dy})
  }

  *cntConect++

  fmt.Println(*cntConect)
  if *cntConect >= 2 {
    sendResult(*answer, *otherAnswer, port, "1")
    *cntConect = 0;
  }
}

func rsvGUIData(cntConect *int, rsvData string, port string, answer *[]Action, otherAnswer *[]Action) {

  parseData := strings.Split(rsvData, ";")

  for i := 0; i < len(parseData) - 1; i++ {
    agentData := strings.Split(parseData[i], " ")

    agentID, _ := strconv.Atoi(agentData[0])
    moveType := agentData[1]
    dx, _ := strconv.Atoi(agentData[2])
    dy, _ := strconv.Atoi(agentData[3])

    *answer = append(*answer, Action{agentID, moveType, dx, dy})
  }

  *cntConect++

  fmt.Println(*cntConect)
  if *cntConect >= 2 {
    sendResult(*otherAnswer, *answer, port, "1")
    *cntConect = 0;
  }
}

func sendResult(solverAnswer []Action, guiAnswer []Action, port string, matchID string) {
  for i := 0; i < len(guiAnswer); i++ {
    for j := 0; j < len(solverAnswer); j++ {
      if solverAnswer[j].AgentID == guiAnswer[i].AgentID {
        solverAnswer[j].Type = guiAnswer[i].Type
        solverAnswer[j].Dx = guiAnswer[i].Dx
        solverAnswer[j].Dy = guiAnswer[i].Dy
        continue;
      }
    }
  }

  sendMoveInform := `{"actions":[`
  for i := 0; i < len(solverAnswer); i++ {
    sendMoveInform += `{"agentID":` + strconv.Itoa(solverAnswer[i].AgentID) + "," + `"type":"` + solverAnswer[i].Type + `",` + `"dx":` + strconv.Itoa(solverAnswer[i].Dx) + "," + `"dy":` + strconv.Itoa(solverAnswer[i].Dy) + "}"
    if i < len(solverAnswer) - 1 {
      sendMoveInform += ","
    }
  }
  sendMoveInform += `]}`

  procon30RequestUrl := "http://localhost:" + port + "/matches/"  + matchID + "/action"
  procon30Token := "procon30_example_token"

  req, err := http.NewRequest(
    "POST",
    procon30RequestUrl,
    bytes.NewBuffer([]byte(sendMoveInform)),
  )

  if err != nil {
    fmt.Println("error")
    return
  }
  req.Header.Set("Authorization", procon30Token)
  req.Header.Set("Content-Type", "application/json")

  client := &http.Client{}
  resp, err := client.Do(req)

  fmt.Println(resp);

  defer resp.Body.Close()
}

func requestFieldData(matchID string, port string, rsvData *[]byte) {
  procon30RequestUrl := "http://localhost:" + port + "/matches/"  + matchID
  procon30Token := "procon30_example_token"

  req, err := http.NewRequest("GET", procon30RequestUrl, nil)
  if err != nil {
    fmt.Println("error")
    return
  }
  req.Header.Set("Authorization", procon30Token)

  client := new(http.Client)
  resp, err := client.Do(req)
  defer resp.Body.Close()

  rsvFieldData, _ := ioutil.ReadAll(resp.Body)

  *rsvData = rsvFieldData
}

func checkArea(field FieldData, teamID int) [][]bool {
  dx := [9]int{0, -1, -1, 0, 1, 1, 1, 0, -1}
  dy := [9]int{0, 0, -1, -1, -1, 0, 1, 1, 1}
  var fl [][]bool

  for i := 0; i < field.Height; i++ {
    tmp := []bool{}
    fl = append(fl, tmp)
    for j := 0; j < field.Width; j++ {
      fl[i] = append(fl[i], false)
    }
  }

  for i := 1; i < field.Height - 1; i++ {
    startArea := false
    var startPos int

    for j := 1; j < field.Width - 1; j++ {
      myTile := 0;

      for k := 1; k < 9; k += 2 {
        if field.Tiled[i + dy[k]][j + dx[k]] == teamID || fl[i + dy[k]][j + dx[k]] {
          if field.Tiled[i][j] != teamID {
            myTile += 1
          }
        }
      }

      if myTile > 1 {
        fl[i][j] = true
        if !startArea {
          startArea = true
          startPos = j
        }
      }

      if field.Tiled[i][j] == teamID && startArea {
        startArea = false
        startPos = j + 1
      }

      if myTile < 2 && startArea {
        for k := startPos; k < j + 1; k++ {
          fl[i][k] = false
        }
        startArea = false
      }
    }
  }


  for i := field.Height - 2; i > 0; i-- {
    for j := field.Width - 2; j > 0; j-- {
      myTile := 0

      for k := 1; k < 9; k += 2 {
        if field.Tiled[i + dy[k]][j + dx[k]] == teamID || fl[i + dy[k]][j + dx[k]] {
          if field.Tiled[i][j] != 1 {
            myTile++
          }
        }
      }
      if myTile < 4 {
        fl[i][j] = false
      }
    }
  }

  areaPoint := 0

  for i := 0; i < field.Height; i++ {
    for j := 0; j < field.Width; j++ {
      if fl[i][j] {
        if field.Points[i][j] < 0 {
          areaPoint += field.Points[i][j] * -1
        } else {
          areaPoint += field.Points[i][j]
        }
      }
    }
  }

  return fl
}

func integrationArea(field FieldData) [][]int {
  var areaPoint [][]int

  for i := 0; i < field.Height; i++ {
    tmp := []int{}
    areaPoint = append(areaPoint, tmp)
    for j := 0; j < field.Width; j++ {
      areaPoint[i] = append(areaPoint[i], 0)
    }
  }

  var rivalArea [][]bool
  myArea := checkArea(field, myTeamID)
  if field.Teams[0].TeamID == myTeamID {
    rivalArea = checkArea(field, field.Teams[1].TeamID)
  } else {
    rivalArea = checkArea(field, field.Teams[1].TeamID)
  }

  for i := 0; i < field.Height; i++ {
    for j := 0; j < field.Width; j++ {
      if myArea[i][j] && rivalArea[i][j] {
        areaPoint[i][j] = 3
      } else if !myArea[i][j] && rivalArea[i][j] {
        areaPoint[i][j] = 2
      } else if myArea[i][j] && !rivalArea[i][j] {
        areaPoint[i][j] = 1
      }
    }
  }

  for i := 0; i < field.Height; i++ {
    fmt.Println(areaPoint[i])
  }

  return areaPoint
}

// Jsonをsolverに送るために変換 {{{
func convertJsonToSendData() string {
  var fieldData FieldData

  var rsvData []byte

  requestFieldData("1", "8080", &rsvData)

  if err := json.Unmarshal(rsvData, &fieldData); err != nil {
    log.Fatal(err)
  }

  var convertData string

  convertData += strconv.Itoa(fieldData.Width)
  convertData += " "

  convertData += strconv.Itoa(fieldData.Height)
  convertData += "\n"

  for i := 0; i < fieldData.Height; i++ {
    for j := 0; j < fieldData.Width; j++ {
      convertData += strconv.Itoa(fieldData.Points[i][j])
      if j != fieldData.Width - 1 {
        convertData += " "
      }
    }
    convertData += ";"
  }
  convertData += "\n"

  convertData += strconv.Itoa(fieldData.StartedAtUnixTime)
  convertData += "\n"

  convertData += strconv.Itoa(fieldData.Turn)
  convertData += "\n"

  for i := 0; i < fieldData.Height; i++ {
    for j := 0; j < fieldData.Width; j++ {
      convertData += strconv.Itoa(fieldData.Tiled[i][j])
      if j != fieldData.Width - 1 {
        convertData += " "
      }
    }
    convertData += ";"
  }
  convertData += "\n"

  convertData += strconv.Itoa(len(fieldData.Teams[0].Agents))
  convertData += "\n"

  if fieldData.Teams[0].TeamID == myTeamID {
    for i := 0; i < 2; i++ {
      convertData += strconv.Itoa(fieldData.Teams[i].TeamID)
      convertData += "\n"

      for j := 0; j < len(fieldData.Teams[i].Agents); j++ {
        convertData += strconv.Itoa(fieldData.Teams[i].Agents[j].AgentID)
        convertData += " "
        convertData += strconv.Itoa(fieldData.Teams[i].Agents[j].X)
        convertData += " "
        convertData += strconv.Itoa(fieldData.Teams[i].Agents[j].Y)
        convertData += ";"
      }
      convertData += "\n"

      convertData += strconv.Itoa(fieldData.Teams[i].TilePoint)
      convertData += " "
      convertData += strconv.Itoa(fieldData.Teams[i].AreaPoint)
      convertData += "\n"
    }
  } else {
    for i := 1; i >= 0; i-- {
      convertData += strconv.Itoa(fieldData.Teams[i].TeamID)
      convertData += "\n"

      for j := 0; j < len(fieldData.Teams[i].Agents); j++ {
        convertData += strconv.Itoa(fieldData.Teams[i].Agents[j].AgentID)
        convertData += " "
        convertData += strconv.Itoa(fieldData.Teams[i].Agents[j].X)
        convertData += " "
        convertData += strconv.Itoa(fieldData.Teams[i].Agents[j].Y)
        convertData += ";"
      }
      convertData += "\n"

      convertData += strconv.Itoa(fieldData.Teams[i].TilePoint)
      convertData += " "
      convertData += strconv.Itoa(fieldData.Teams[i].AreaPoint)
      convertData += "\n"
    }
  }
  convertData += maxTurn + "\n"

  return convertData
}
//}}}
// JsonをGUIに送るために変換 {{{
func convertJsonToSendDataForGUI() string {
  var fieldData FieldData

  var rsvData []byte

  requestFieldData("1", "8080", &rsvData)

  fmt.Println("aaa");
  if err := json.Unmarshal(rsvData, &fieldData); err != nil {
    log.Fatal(err)
  }

  area := integrationArea(fieldData)


  var convertData string

  convertData += strconv.Itoa(fieldData.Width)
  convertData += " "

  convertData += strconv.Itoa(fieldData.Height)
  convertData += "\n"

  for i := 0; i < fieldData.Height; i++ {
    for j := 0; j < fieldData.Width; j++ {
      convertData += strconv.Itoa(fieldData.Points[i][j])
      if j != fieldData.Width - 1 {
        convertData += " "
      }
    }
    convertData += ";"
  }
  convertData += "\n"

  convertData += strconv.Itoa(fieldData.StartedAtUnixTime)
  convertData += "\n"

  convertData += strconv.Itoa(fieldData.Turn)
  convertData += "\n"

  for i := 0; i < fieldData.Height; i++ {
    for j := 0; j < fieldData.Width; j++ {
      convertData += strconv.Itoa(fieldData.Tiled[i][j])
      if j != fieldData.Width - 1 {
        convertData += " "
      }
    }
    convertData += ";"
  }
  convertData += "\n"

  convertData += strconv.Itoa(len(fieldData.Teams[0].Agents))
  convertData += "\n"

  if fieldData.Teams[0].TeamID == myTeamID {
    for i := 0; i < 2; i++ {
      convertData += strconv.Itoa(fieldData.Teams[i].TeamID)
      convertData += "\n"

      for j := 0; j < len(fieldData.Teams[i].Agents); j++ {
        convertData += strconv.Itoa(fieldData.Teams[i].Agents[j].AgentID)
        convertData += " "
        convertData += strconv.Itoa(fieldData.Teams[i].Agents[j].X)
        convertData += " "
        convertData += strconv.Itoa(fieldData.Teams[i].Agents[j].Y)
        convertData += ";"
      }
      convertData += "\n"

      convertData += strconv.Itoa(fieldData.Teams[i].TilePoint)
      convertData += " "
      convertData += strconv.Itoa(fieldData.Teams[i].AreaPoint)
      convertData += "\n"
    }
  } else {
    for i := 1; i >= 0; i-- {
      convertData += strconv.Itoa(fieldData.Teams[i].TeamID)
      convertData += "\n"

      for j := 0; j < len(fieldData.Teams[i].Agents); j++ {
        convertData += strconv.Itoa(fieldData.Teams[i].Agents[j].AgentID)
        convertData += " "
        convertData += strconv.Itoa(fieldData.Teams[i].Agents[j].X)
        convertData += " "
        convertData += strconv.Itoa(fieldData.Teams[i].Agents[j].Y)
        convertData += ";"
      }
      convertData += "\n"

      convertData += strconv.Itoa(fieldData.Teams[i].TilePoint)
      convertData += " "
      convertData += strconv.Itoa(fieldData.Teams[i].AreaPoint)
      convertData += "\n"
    }
  }
  convertData += maxTurn + "\n"

  fmt.Println("aaa")
  for i := 0; i < fieldData.Height; i++ {
    for j := 0; j < fieldData.Width; j++ {
      convertData += strconv.Itoa(area[i][j])
      if j != fieldData.Width - 1 {
        convertData += " "
      }
    }
    convertData += ";"
  }
  convertData += "\n"

  fmt.Println(convertData)

  return convertData
}
//}}}
