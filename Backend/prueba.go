package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"
)

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

// ANIO,MES,DIA,CODIGO,TOT_VEH_PAG,TOT_VEH_EXON,SENT_COBRO,FLUJO_VEH
var dataset_entrenamiento = []PeajeData{}
var K int = 1

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
	fmt.Println(req.FormValue("anio"))
	anio, _ := strconv.ParseFloat(req.FormValue("anio"), 32)
	mes, _ := strconv.ParseFloat(req.FormValue("mes"), 64)
	dia, _ := strconv.ParseFloat(req.FormValue("dia"), 64)
	codigo, _ := strconv.ParseFloat(req.FormValue("codigo"), 64)
	sent_cobro, _ := strconv.ParseFloat(req.FormValue("sent_cobro"), 64)
	fmt.Println(anio)
	var test PeajeData = PeajeData{
		Anio:       anio,
		Mes:        mes,
		Dia:        dia,
		Codigo:     codigo,
		Sent_cobro: sent_cobro,
	}
	fmt.Println(test)
	res.Header().Set("Access-Control-Allow-Origin", "*")
	res.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	res.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Authorization")
	res.Header().Set("Content-Type", "application/json")

	var flujo_veh chan int64 = make(chan int64)

	go algoritmo_knn(dataset_entrenamiento, test, flujo_veh)
	test.Flujo_veh = <-flujo_veh
	json.NewEncoder(res).Encode(test)
}
func handlerRequest() {
	http.HandleFunc("/flujovehiculo", obtenerFlujoVehicular)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
func main() {
	url := "https://raw.githubusercontent.com/richhardd/dataset-progconc/main/dataset.csv"
	Archivo, _ := leerCSV(url)
	convertirData(Archivo)
	handlerRequest()
	fmt.Println(dataset_entrenamiento)
}
