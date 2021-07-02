package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

//VARIABLES GLOBALES
var dataset_entrenamiento = []PeajeData{}
var K int = 1
var direccion_nodo string
var direcciones []string
var puedeIniciar chan bool
var chMyInfo chan MyInfo

type Info struct {
	Tipo     string //tipo de mensaje
	NodeNum  int    //nro de ticket del nodo que solicita el permiso
	NodeAddr string //ip del nodo que solicita el permiso
}

type MyInfo struct {
	contMsg  int  //control de los mensajes recibidos
	first    bool //si le toca acceder a la sección crítica
	nextNum  int  //proximo ticket
	nextAddr string
}

const (
	numero_nodo       = 1
	puerto_registro   = 8000
	puerto_notifica   = 8001
	puerto_proceso_hp = 8002
	puerto_solicitud  = 8003
)

type Request struct {
	Dates            []Date  `json:"dates"`
	TollCode         string  `json:"TollCode"`
	PaymentDirection float64 `json:"ParmentDirection"`
}
type Response struct {
	Count   int      `json:"count"`
	Results []Result `json:"results"`
}
type Result struct {
	Date  string  `json:"date"`
	Score float64 `json:"score"`
}
type Date struct {
	Day   float64 `json:"day"`
	Month float64 `json:"month"`
	Year  float64 `json:"year"`
}
type PeajeData struct {
	Anio         float64
	Mes          float64
	Dia          float64
	Codigo       float64
	Tot_veh_pag  float64
	Tot_veh_exon float64
	Sent_cobro   float64
	Flujo_veh    int64
	ditancia     float64
}

//funciones de nodo
func iplocal() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Print(fmt.Errorf("localAddress: %v\n", err.Error()))
		return "127.0.0.1"
	}
	for _, oiface := range ifaces {
		if strings.HasPrefix(oiface.Name, "Wi-Fi") {
			//if strings.HasPrefix(oiface.Name, "ens33") {
			addrs, err := oiface.Addrs()
			if err != nil {
				log.Print(fmt.Errorf("localAddress: %v\n", err.Error()))
				continue
			}
			for _, dir := range addrs {
				switch d := dir.(type) {
				case *net.IPNet:
					if strings.HasPrefix(d.IP.String(), "192") {
						return d.IP.String()
					}
				}
			}
		}
	}
	return "127.0.0.1"
}
func Notificar(direccion, ip string) {
	//formato del host remoto
	hostremoto := fmt.Sprintf("%s:%d", direccion, puerto_notifica)
	//establecer conexion con host remoto
	conn, _ := net.Dial("tcp", hostremoto)
	defer conn.Close()
	//enviar el mensaje IP al nodo remoto
	fmt.Fprintf(conn, "%s\n", ip)
}
func NotificarTodos(ip string) {
	//recorriendo la bitacora
	for _, direccion := range direcciones {
		Notificar(direccion, ip)
	}
}
func manejadorRegistro(conn net.Conn) {
	defer conn.Close()
	//leer la ip que llega como mensaje de la solicitud
	bufferIn := bufio.NewReader(conn)
	msgIP, _ := bufferIn.ReadString('\n')
	msgIP = strings.TrimSpace(msgIP)
	//notificar a todos las ips de la bitacora
	//codificar el mensaje en formato json
	bytesDirecciones, _ := json.Marshal(direcciones)
	//enviar un msg de respuesta al nuevo nodo con la bitacora actual
	fmt.Fprintf(conn, "%s\n", string(bytesDirecciones))
	//enviar a los IPs
	NotificarTodos(msgIP)
	//actualizar su bitacora con la nueva direccion
	direcciones = append(direcciones, msgIP)
	fmt.Println(direcciones)
}
func AtenderRegistroCliente() {
	hostlocal := fmt.Sprintf("%s:%d", direccion_nodo, puerto_registro)
	//modo escucha
	ln, _ := net.Listen("tcp", hostlocal)
	defer ln.Close()
	//atencion de solicitudes
	for {
		conn, _ := ln.Accept() //acepta conexiones
		go manejadorRegistro(conn)
	}
}
func ManejadorNotificacion(conn net.Conn) {
	defer conn.Close()
	//leer el mensaje enviado
	bufferIn := bufio.NewReader(conn)
	msgIP, _ := bufferIn.ReadString('\n')
	msgIP = strings.TrimSpace(msgIP)
	//agregamos la ip del nuevo nodo a la bitacora actual
	direcciones = append(direcciones, msgIP)
	fmt.Println(direcciones)
}
func AtenderNofificaciones() {
	//modo escucha
	hostlocal := fmt.Sprintf("%s:%d", direccion_nodo, puerto_notifica)
	ln, _ := net.Listen("tcp", hostlocal)
	defer ln.Close()
	for {
		conn, _ := ln.Accept()
		go ManejadorNotificacion(conn)
	}
}
func RegistrarCliente(ipremoto string) {
	//solicitud a un nodo de la red ipremoto
	hostremoto := fmt.Sprintf("%s:%d", ipremoto, puerto_registro) //IP:puerto
	//realizar la llamada de conexion al host remoto
	conn, _ := net.Dial("tcp", hostremoto)
	defer conn.Close()
	//enviar IP del nuevo nodo
	fmt.Fprintf(conn, "%s\n", direccion_nodo)
	//recibe bitacora del host remoto
	bufferIn := bufio.NewReader(conn)
	msgDirecciones, _ := bufferIn.ReadString('\n')
	//decodificamos (json) el mensaje recibido
	var auxDirecciones []string
	json.Unmarshal([]byte(msgDirecciones), &auxDirecciones)
	direcciones = append(auxDirecciones, ipremoto) //agregar la ip remota a bytes
	fmt.Println(direcciones)                       //imprimir bitacora de clientes

}
func ManejadorEnvioSolicitudes(addr string, msgInfo Info) {
	addr = strings.TrimSpace(addr)
	remoteHost := fmt.Sprintf("%s:%d", addr, puerto_solicitud)
	//llamada al ip remoto
	conn, _ := net.Dial("tcp", remoteHost)
	defer conn.Close()
	//notifico
	//codificar el mensaje
	bytesMsg, _ := json.Marshal(msgInfo)
	fmt.Fprintln(conn, string(bytesMsg)) //enviando el mensaje serializado en string
}

//funciones modelo de ML
func dist_euclidiana(modelo, test PeajeData, distasync chan float64) {
	mes_dist := math.Pow(test.Mes-modelo.Mes, 2)
	dia_dist := math.Pow(test.Dia-modelo.Dia, 2)
	sent_cobro_dist := math.Pow(test.Sent_cobro-modelo.Sent_cobro, 2)
	codigo_dis := math.Pow(test.Codigo-modelo.Codigo, 2)
	response := math.Sqrt(mes_dist + dia_dist + sent_cobro_dist + codigo_dis)
	distasync <- response
}
func algoritmo_knn(modelo []PeajeData, test PeajeData, respuesta chan int64) {
	var modelo_validacion []PeajeData
	distasync := make(chan float64)
	for _, one := range modelo {
		go dist_euclidiana(one, test, distasync)
		one.ditancia = <-distasync
		modelo_validacion = append(modelo_validacion, one)
	}
	sort.SliceStable(modelo_validacion, func(x, y int) bool {
		comparacion := modelo_validacion[x].ditancia < modelo_validacion[y].ditancia
		return comparacion
	})
	respuesta <- modelo_validacion[:K][0].Flujo_veh
}
func convertirData(archivo [][]string) {
	for i := 1; i < len(archivo); i++ {
		anio, _ := strconv.ParseFloat(archivo[i][0], 64)
		mes, _ := strconv.ParseFloat(archivo[i][1], 64)
		dia, _ := strconv.ParseFloat(archivo[i][2], 64)
		codigo, _ := strconv.ParseFloat(archivo[i][3], 64)
		tot_veh_pag, _ := strconv.ParseFloat(archivo[i][4], 64)
		tot_veh_exon, _ := strconv.ParseFloat(archivo[i][5], 64)
		sent_cobro, _ := strconv.ParseFloat(archivo[i][6], 64)
		flujo_veh, _ := strconv.ParseInt(archivo[i][7], 32, 32)

		var temp PeajeData = PeajeData{
			Anio:         anio,
			Mes:          mes,
			Dia:          dia,
			Codigo:       codigo,
			Tot_veh_pag:  tot_veh_pag,
			Tot_veh_exon: tot_veh_exon,
			Sent_cobro:   sent_cobro,
			Flujo_veh:    flujo_veh,
		}
		dataset_entrenamiento = append(dataset_entrenamiento, temp)
	}
}
func leerCSV(url string) ([][]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	reader := csv.NewReader(resp.Body)
	reader.Comma = ','
	data, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	return data, nil
}
func obtenerFlujoVehicular(res http.ResponseWriter, req *http.Request) {
	request := Request{}
	err := json.NewDecoder(req.Body).Decode(&request)
	if err != nil {
		panic(err)
	}
	fmt.Println("request", request.PaymentDirection)
	fmt.Println("request", request.TollCode)
	fmt.Println("request", len(request.Dates))
	fmt.Println(req.FormValue("anio"))
	// anio, _ := strconv.ParseFloat(req.FormValue("anio"), 32)
	// mes, _ := strconv.ParseFloat(req.FormValue("mes"), 64)
	// dia, _ := strconv.ParseFloat(req.FormValue("dia"), 64)
	// codigo, _ := strconv.ParseFloat(req.FormValue("codigo"), 64)
	// sent_cobro, _ := strconv.ParseFloat(req.FormValue("sent_cobro"), 64)
	// fmt.Println(anio)
	// var test PeajeData = PeajeData{
	// 	Anio:       anio,
	// 	Mes:        mes,
	// 	Dia:        dia,
	// 	Codigo:     codigo,
	// 	Sent_cobro: sent_cobro,
	// }

	res.Header().Set("Access-Control-Allow-Origin", "*")
	res.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	res.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Authorization")
	res.Header().Set("Content-Type", "application/json")
	response := Response{}
	var test Result = Result{
		Date:  "22-02-2021",
		Score: 2,
	}
	response.Results = append(response.Results, test)
	response.Count = len(response.Results)
	// var flujo_veh chan int64 = make(chan int64)

	// go algoritmo_knn(dataset_entrenamiento, test, flujo_veh)
	// test.Flujo_veh = <-flujo_veh
	json.NewEncoder(res).Encode(response)
}
func handlerRequest() {
	http.HandleFunc("/flujovehiculo", obtenerFlujoVehicular)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
func lecturaDataset() {
	url := "https://raw.githubusercontent.com/richhardd/dataset-progconc/main/dataset.csv"
	Archivo, _ := leerCSV(url)
	convertirData(Archivo)
}
func main() {
	lecturaDataset()
	direccion_nodo = iplocal()
	// fmt.Println("IP: ", direccion_nodo)
	// go AtenderRegistroCliente()

	// bufferIn := bufio.NewReader(os.Stdin) //ingreso por consola
	// fmt.Println("ingrese la ip del host para solicitud: ")
	// ipremoto, _ := bufferIn.ReadString('\n')
	// ipremoto = strings.TrimSpace(ipremoto)
	// if ipremoto != "" {
	// 	//solo para nuevos nodos
	// 	RegistrarCliente(ipremoto)
	// }
	// go AtenderNofificaciones()
	// puedeIniciar = make(chan bool)
	// chMyInfo = make(chan MyInfo)
	// go func() {
	// 	chMyInfo <- MyInfo{0, true, 1000001, ""}
	// }()

	handlerRequest()
}
