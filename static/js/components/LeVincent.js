
import { ref, reactive, onMounted } from 'vue'


// id generator for mapComponents
let mapCount = 0
function nextMapId() {
  return ++mapCount
}


export const RootComponent = {

  setup() {

    // map data properties must be declared at component creation 
    // filled asynchronously by fetchMapdata() later
    const mapData = reactive({
      'name': '',
      'idtech': '',
      'taxonomy': '',
      'bbox': {},
      'subzones': new Array(),
      'prevs': {},
    })

    // selection of displayed data
    const selections = reactive({
      tooltipsEnabled: false,
      activeWeather: String("default")
      //activeTimespan: String("")
    })

    const breadcrumb = reactive([
      { name: "pays", path: "/path_france" },
      { name: "région", path: "/path_region" },
      { name: "dept", path: "/path_dept" },
    ])

    async function fetchMapdata() {
      console.log("enter fetchMapdata()")
      const res = await fetch("/france/data")
      const data = await res.json()

      mapData.name = data.name
      mapData.idtech = data.idtech
      mapData.taxonomy = data.taxonomy
      mapData.bbox = data.bbox
      mapData.subzones = data.subzones
      mapData.prevs = data.prevs
      console.log("exit fetchMapdata()")
    }

    // get Prevs when static page is loaded
    onMounted(fetchMapdata)

    // only returned items are available in template
    return {
      mapData, selections, breadcrumb
    }
  },


  template: /*html*/ `
  <header>
  <Breadcrumb :breadcrumb="breadcrumb"/>
  <section class="selecteurs">
  <p>mapname={{mapData.name}}</p>
  <!--  <DataPicker/>
    <div>
    <TimespanPicker/>
    <TooltipsToggler/>
  </div> -->
<!--  <highchart-graph v-if="display_graph"></highchart-graph> -->
  </section>
</header>
<!--    <h2 style="color: rgb(43, 70, 226);">2024-08-18 : Tests en cours ...<P></P> </h2> -->
<main class="content">
  <MapGridComponent 
    :data="mapData" 
    :selections="selections" />
</main>`,
}

export const Breadcrumb = {
  props: {
    breadcrumb: Array
  },

  template: /*html*/`
<nav class="topnav">
  Navigation : 
  <ul>
  <li v-for="item in breadcrumb">
    <a :href="item.path">{{item.name}}</a>
  </li>
  </ul>
</nav>`
}

export const DataPicker = {

  template: /*html*/`
  <p>DataPicker component</p>`
}

export const TooltipsToggler = {

  template: /*html*/`
  <p>TooltipsToggler component</p>`
}

export const TimespanPicker = {

  template: /*html*/`
  <p>TimespanPicker component</p>`
}


export const MapGridComponent = {

  props: {
    data: Object,
    selections: Object,
  },

  setup(props) {

    const displayedJours = (() => {
      //console.log( "displayedJours() typeof props.data.prevs is " + typeof props.data.prevs )

      const ret = []
      if ( typeof props.data.prevs === 'undefined') {
        return ret
      }
      for (var i = -1; i < 3; i++) {
        if ( Object.hasOwn(props.data.prevs, i)) {
          ret.push(props.data.prevs[i])
        }
      }
/*    for (let jour in props.data.prevs) {
        let n = parseInt(jour)
        //if (n < 3 && n >= 0) {
          if (n < 3) {
          ret.push(props.data.prevs[jour])
        }
      }*/
      return ret
    })

    return { displayedJours }
  },

  template: /*html*/`
<div class="map_grid_container">
  <div>
    <p>  before MapRowComponent </p> 
    <MapRowComponent
    v-for="(jour, idx) in displayedJours()"
    :key="idx"
    :prevsDuJour="jour"
    :data="data"
    :selections="selections"/>
  </div>
</div>`
}


export const MapRowComponent = {

  props: {
    prevsDuJour: Array,
    data: Object,
    selections: Object,
  },

  setup(props) {
  },

  template: /*html*/`
  <p> ... inside MapRowComponent ... </p>
  <MapComponent v-for="(prev, idx) in prevsDuJour"
  :key="idx"
  :prev="prev"
  :data="data"
  :selections="selections">
</MapComponent>`
}


export const MapComponent = {
  props: {
    prev: Object,
    data: Object,
    selections: Object,
  },

  setup(props) {

    const weatherNames = new Map([
      ["default", "Default"],
    ])
    /*    ["matin", "Matin"],
          ["après-midi", "Aprèm"],
          ["soirée", "Soir"],
          ["nuit", "Nuit"],*/

    let _map_id = 0
    const mapId = function () {
      if( _map_id == 0 ) {
        _map_id = nextMapId()
      }
      return String(_map_id)
    }

    const mapTitle = function () {
      let weather = (typeof props.selections != null) ? props.selections.activeWeather : ""
      let moment = (props.prev != null)  ? props.prev.echeance : ""
      return moment + ' - ' + weather
    }

    // leaflet.Map object cannot be created in setup() 
    // because DOM element does not exist before onMounted()
    // we need a reference in component instance to control tooltips and displayed data
    let lMap = null

    // keep references to markers for update/deletion
    let markers = []

    function initMap() {
      //  when timespan changes, components are cached/re-used by v-for algorithm
      // so just skip initMap because map and subzones do not change.
      // if (this.map) 
      //  return true;

      // format bounds in a leaflet-specific object
      let bbox = props.data.bbox
      let bounds = L.latLngBounds([[bbox.s, bbox.w], [bbox.n, bbox.e]])

      // setup main leaflet object
      lMap = L.map(mapId(), {
        center: bounds.center,
        fullscreenControl: true,
        cursor: true,
        scrollWheelZoom: false,
        zoomSnap: 1e-4,
        zoomDelta: .1,
        zoomControl: false,
        dragging: false,
        tap: false,
        maxBoundsViscosity: 1,
        keyboard: false,
        doubleClickZoom: false,
        attributionControl: false,
      })

      // add SVG map background
      let overlay = L.imageOverlay(svgPath(), bounds)
      lMap.addLayer(overlay)
      lMap.setMaxBounds(bounds)
      lMap.fitBounds(bounds)
      lMap.setMinZoom(lMap.getBoundsZoom(bounds, true))

      drawSubzones()

      // todo: add updated date in "attributions"

      // todo : reactive markers on props.selections
      updateMarkers()
    }

    function svgPath() {
      var img = new Image
      img.src = '/france/svg'
      return img
    }


    function drawSubzones() {

      const szPane = lMap.createPane('subzones')

      props.data.subzones.forEach((sz) => {
        let path = 'todo_subzone_path' //sz.properties.prop_custom.path
        //let nom = sz.properties.prop_custom.name 
        L.geoJSON(sz, {
          color: "transparent",
          fillColor: "transparent",
          weight: 3,
          pane: 'subzones',
          onEachFeature: function (feature, layer) {
            // layer.bindTooltip(nom, { direction: "auto" });
            layer.on("mouseover", function () {
              this.setStyle({ color: "#FFF", fillColor: "transparent" })
              layer.openPopup()
            }),
            layer.on("mouseout", function () {
              this.setStyle({ color: "transparent", fillColor: "transparent" })
              layer.closePopup()
            }),
            layer.on("click", function () {
              window.location = path
            })
          }
        }
        ).addTo(lMap);
      })
    }

    function updateMarkers() {
      let pois = props.prev && props.prev.prevs
      if (pois) {
          removeMarkers();   // todo : inline
          pois.forEach( createMarker );

        // TODO
        // this.updateControl.setPrefix ("Màj : "+this.get_prevs.updated );
      }
    }

    function removeMarkers() {
      while ( markers.length > 0 ) {
        lMap.removeLayer(markers.pop());
      }
    }

    function createMarker(poi, idx, all_prevs) {

      // local aliases
      const prev = poi.prev
      const daily = poi.daily

      // accumulate marker data for current poi
      const mark = {}

      // default marker
      mark.disabled = false;
      mark.icon = prev.weather_icon;
      // TODO: fallback on daily_weather_icon if weather_icon == null
      mark.txt = null

      if ( prev.T !== null ) {
        // donnée court-terme en priorité si disponibles
        mark.T = Math.round(prev.T)
        mark.txt = mark.T + "°"
      } else if (daily.T_min !== null && daily.T_max !==null) {
        // donnée long-terme (dailies) 
        mark.txt = ' <span class="tmin">' + Math.round(daily.T_min) + '°</span>' +
                  '/<span class="tmax">' + Math.round(daily.T_max) + '°</span>'
      } else {
          // pas de marker si temperature indisponible
          mark.disabled = true
      }

      mark.coords = poi.coords
      mark.titre = poi.titre
      // TODO: fallback on daily_weather_desc if weather_desc == null
      mark.desc = prev.weather_description
      mark.Tmin = Math.round(prev.T_min)
      mark.Tmax = Math.round(prev.T_max)

      addMarker( mark )
    }


    function addMarker(mark) {
      if( mark.disabled) {
        return 
      }
//      console.log(["addMarker()", mark])
      let icon_width = 40
      if ( props.selections.activeWeather == 'vent') {
        icon_width = 25    // icones du vent plus petites
      }

      // style pour le texte température min/max (qui contient un slash)
      let icon_text_style = ""
      /*
      if( ("string" == typeof e.icon_text || e.icon_text instanceof String) && 
              e.icon_text.indexOf("/") > -1 ) {
          (icon_text_style = "font-size: 12px;");  */

      let elt_a = '<a>'+
      '<img src="/pictos/' + mark.icon + '" ' +
      'alt="' + mark.desc + '" ' +
      'title="' + mark.titre + '" ' +
      'class="icon shape-weather" ' +
      'style="width: ' + icon_width + 'px"/>'

      if( mark.txt && "" != mark.txt && "NaN°" != mark.txt ) {
        elt_a += '<span class="icon_text" style="' + icon_text_style + '">' +
                  mark.txt + '</span>'
      }
      elt_a += "</a>"

      let mark_opts = {
        icon : L.divIcon({
          html: elt_a,
          className: "iconMap-1",
          iconSize: [icon_width, icon_width],
          iconAnchor: [icon_width / 2, icon_width / 2]
        })
      }

      // GeoJSON order : lng,lat
      // Leaflet API order : lat,lng
      //let coords = L.LatLng( mark.coords[1], mark.coords[0])
      let lMark = L.marker([mark.coords[1], mark.coords[0]], mark_opts).addTo(lMap);

      // keep a reference for marker deletion
      markers.push(lMark)
    }
/*
    function markerPosition(coords) {
      let bbox = props.data.bbox
      let pos_h = (coords[0] > (bbox.o + bbox.e)/2 ) ? 'Right' : 'Left'
      let pos_v = (coords[1] > (bbox.n + bbox.s)/2 ) ? 'Top' : 'Bottom'
      return pos_v+' '+pos_h
    }
*/

    onMounted(initMap)
    return { mapTitle, mapId }
  },

  template: /*html*/`
<div class="map_grid_item">
  <div class="titre_carte"> {{ mapTitle() }} </div>
  <div :id="mapId()" class="map_component"></div>
</div>`

}



