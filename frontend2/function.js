let mes =  document.getElementById("mes")
let dia =  document.getElementById("dia")
let codigo  = document.getElementById("codigo")
let sent_cobro = document.getElementById("sent_cobro")



let submitButton = document.getElementById("submit-btn")
let resultado = document.getElementById("resultado_pred")


submitButton.addEventListener("click", () => {

    fetch(`http://localhost:8080/flujovehiculo?anio=2021&mes=${mes.value}&dia=${dia.value}&codigo=${codigo.value}&sent_cobro=${sent_cobro.value}`)
    .then((s) => s.json())
    .then((data) => {
        resultado.textContent= data.Flujo_veh
    })
}, false)