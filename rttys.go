package main

import (
    "flag"
    "fmt"
    "log"
    "sync"
    "time"
    "errors"
    "strconv"
    "net/http"
    "math/rand"
    "log/syslog"
    "crypto/md5"
    "encoding/hex"
    "encoding/json"
    "github.com/gorilla/websocket"

    _ "github.com/zhaojh329/rttys/statik"
    "github.com/rakyll/statik/fs"
)

var cross *bool
var slog *log.Logger
var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
}
var dev2wsConnection = make(map[string] *wsConnection)
var sid2wsConnection = make(map[string] *wsConnection)

const (
    FromDevice = 0
    FromBrowser = 1
)

type RttyFrame struct {
    Type string `json:"type"`
    SID string `json:"sid"`
    Data string `json:"data"`
    Err string `json:"err"`
}

/* Used for /list */
type DeviceInfo struct {
    ID string `json:"id"`
    Uptime int64 `json:"uptime"`
    Description string `json:"description"`
}

type wsMessage struct {
    msgType int
    data []byte
}

type wsConnection struct {
    from int
    contime int64        /* connect time */
    did string
    sid string          /* only valid for from browser */
    active int          /* only valid for from device */
    description string  /* only valid for from device */
    ws *websocket.Conn
    inChan chan *wsMessage
    outChan chan *wsMessage

    mutex sync.Mutex
    isClosed bool
    closeChan chan byte
}

func generateSID(did string) string {
    md5Ctx := md5.New()
    md5Ctx.Write([]byte(did + strconv.FormatFloat(rand.Float64(), 'e', 6, 32)))
    cipherStr := md5Ctx.Sum(nil)
    return hex.EncodeToString(cipherStr)
}

func (wsConn *wsConnection)wsClose() {
    if wsConn.from == FromBrowser {
        if devCon, ok := dev2wsConnection[wsConn.did]; ok {
            f := &RttyFrame{Type: "logout", SID: wsConn.sid}
            js, _ := json.Marshal(f)
            devCon.wsWrite(websocket.TextMessage, js)
        }
    } else {
        delete(dev2wsConnection, wsConn.did)
    }

    wsConn.ws.Close()

    defer wsConn.mutex.Unlock()

    wsConn.mutex.Lock()
    if !wsConn.isClosed {
        wsConn.isClosed = true
        close(wsConn.closeChan)
    }
}

func (wsConn *wsConnection)wsWriteLoop() {
    for {
        select {
        case msg := <- wsConn.outChan:
            if err := wsConn.ws.WriteMessage(msg.msgType, msg.data); err != nil {
                goto error
            }
        case <- wsConn.closeChan:
            goto closed
        }
    }
error:
    wsConn.wsClose()
closed:
}

func (wsConn *wsConnection)wsReadLoop() {
    for {
        msgType, data, err := wsConn.ws.ReadMessage()
        if err != nil {
            goto error
        }
        req := &wsMessage{msgType, data}
        select {
        case wsConn.inChan <- req:
        case <- wsConn.closeChan:
            goto closed
        }
    }
error:
    wsConn.wsClose()
closed:
}

func (wsConn *wsConnection)wsWrite(messageType int, data []byte) error {
    select {
    case wsConn.outChan <- &wsMessage{messageType, data}:
    case <- wsConn.closeChan:
        return errors.New("websocket closed")
    }
    return nil
}

func (wsConn *wsConnection)wsRead() (*wsMessage, error) {
    select {
    case msg := <- wsConn.inChan:
        return msg, nil
    case <- wsConn.closeChan:
    }
    return nil, errors.New("websocket closed")
}

func (wsConn *wsConnection)procLoop() {
    for {
        msg, err := wsConn.wsRead()
        if err != nil {
            break
        }

        if msg.msgType == websocket.TextMessage {
            f := &RttyFrame{}
            json.Unmarshal(msg.data, f)

            if wsConn.from == FromDevice {
                if f.Type == "ping" {
                    wsConn.active = 3
                    f := &RttyFrame{Type: "pong"}
                    js, _ := json.Marshal(f)
                    wsConn.wsWrite(websocket.TextMessage, js)
                } else if f.Type == "data" {
                    if bwCon, ok := sid2wsConnection[f.SID]; ok {
                        bwCon.wsWrite(websocket.TextMessage, msg.data)
                    }
                } else if f.Type == "logout" {
                    if bwCon, ok := sid2wsConnection[f.SID]; ok {
                        bwCon.wsClose();
                    }
                }
            } else {
                if f.Type == "data" || f.Type == "upfile" {
                    if devCon, ok := dev2wsConnection[wsConn.did]; ok {
                        devCon.wsWrite(msg.msgType, msg.data)
                    }   
                }
            }
        } else if msg.msgType == websocket.BinaryMessage {
            if wsConn.from == FromBrowser {
                if devCon, ok := dev2wsConnection[wsConn.did]; ok {
                    devCon.wsWrite(msg.msgType, msg.data)
                }
            }
        }
    }
}

func serveWs(w http.ResponseWriter, r *http.Request) {
    path := r.URL.Path
    description := r.URL.Query().Get("des")
    did := r.URL.Query().Get("did")
    if did == "" {
        return
    }

    ws, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        slog.Println("upgrade:", err)
        return
    }

    wsConn := &wsConnection{
        from: FromDevice,
        contime: time.Now().Unix(),
        did: did,
        ws: ws,
        inChan: make(chan *wsMessage, 1000),
        outChan: make(chan *wsMessage, 1000),
        closeChan: make(chan byte),
        isClosed: false,
    }

    if path == "/ws/device" {
        if _, ok := dev2wsConnection[did]; ok {
            f := &RttyFrame{Type: "add", Err: "ID conflicts"}
            js, _ := json.Marshal(f)
            ws.WriteMessage(websocket.TextMessage, js)
            ws.Close()
            return
        }
        wsConn.description = description
        wsConn.active = 3
        dev2wsConnection[did] = wsConn
        slog.Println("New Device:", did)
    } else {
        wsConn.from = FromBrowser

        f := RttyFrame{Type: "login"}
        devCon, ok := dev2wsConnection[did]
        if !ok {
            f.Err = "Device off-line"
            js, _ := json.Marshal(f)
            ws.WriteMessage(websocket.TextMessage, js)
            ws.Close()
        } else {
            /* Login */
            sid := generateSID(did)
            sid2wsConnection[sid] = wsConn
            wsConn.sid = sid
            f.SID = sid
            js, _ := json.Marshal(f)
            ws.WriteMessage(websocket.TextMessage, js)
            devCon.wsWrite(websocket.TextMessage, js)
        }
    }

    go wsConn.procLoop()
    go wsConn.wsReadLoop()
    go wsConn.wsWriteLoop()
}

func handlerList(w http.ResponseWriter, r *http.Request) {
    devs := make([]DeviceInfo, 0)
    for k, con := range dev2wsConnection {
        d := DeviceInfo{k, time.Now().Unix() - con.contime, con.description}
        devs = append(devs, d)
    }

    js, _ := json.Marshal(devs)
    if *cross {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
        w.Header().Set("content-type", "application/json")
    }
    fmt.Fprintf(w, "%s", js)
}

func flushDevice() {
    for {
        for _, con := range dev2wsConnection {
            con.active--
            if con.active == 0 {
                con.wsClose()
            }
        }
        time.Sleep(5 * time.Second)
    }
}

func main() {
    port := flag.Int("port", 5912, "http service port")
    cert := flag.String("cert", "", "certFile Path")
    key := flag.String("key", "", "keyFile Path")
    cross = flag.Bool("cross", false, "Allow Cross domain")

    flag.Parse()

    rand.Seed(time.Now().Unix())

    _slog, err := syslog.New(syslog.LOG_INFO, "rttys")
    if err != nil {
        log.Fatal(err)
        return
    }
    defer _slog.Close()
    slog = log.New(_slog, "", log.Lshortfile | log.LstdFlags)

    statikFS, err := fs.New()
    if err != nil {
        slog.Fatal(err)
        return
    }

    go flushDevice()

    http.HandleFunc("/ws/device", serveWs)
    http.HandleFunc("/ws/browser", serveWs)
    http.HandleFunc("/list", handlerList)
    http.Handle("/", http.FileServer(statikFS))

    if *cert != "" && *key != "" {
        slog.Println("Listen on: ", *port, "SSL on")
        slog.Fatal(http.ListenAndServeTLS(":" + strconv.Itoa(*port), *cert, *key, nil))
    } else {
        slog.Println("Listen on: ", *port, "SSL off")
        slog.Fatal(http.ListenAndServe(":" + strconv.Itoa(*port), nil))
    }
}
