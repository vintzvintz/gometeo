
import { ref, reactive, watch, onMounted } from 'vue'


// id generator for mapComponents
let mapCount = 0
function nextMapId() {
  return ++mapCount
}

const weatherList = {
  "default": {
    text: "default",
  },
  "prev": {
    text: "Prévisions",
  },
  "vent": {
    text: "Vent",
  },
  "ress": {
    text: "Ressenti",
  },
  "humi": {
    text: "Humidité",
  },
  "psea": {
    text: "Pression",
  },
}

const weatherDisplayOrder = [
  "prev", "vent", "ress", "humi", "psea",
]


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
      activeWeather: "prev"
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

    // callback when WeatherPicker emits a 'weatherSelected' event
    function onWeatherSelected(id) {
      selections.activeWeather = id   // reactive
    }

    // get Prevs when static page is loaded
    onMounted(fetchMapdata)

    // only returned items are available in template
    return {
      mapData, selections, onWeatherSelected, breadcrumb
    }
  },

  template: /*html*/ `
  <header>
  <Breadcrumb :breadcrumb="breadcrumb"/>
  <section class="selecteurs">
  <p>mapname={{mapData.name}}</p>
   <WeatherPicker 
   :activeWeather="selections.activeWeather"
   @weatherSelected="onWeatherSelected" />
<!--    <div>
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

export const WeatherPicker = {

  emits: ['weatherSelected'],

  props: {
    activeWeather: String,
  },

  setup() {
    return { weatherList, weatherDisplayOrder }
  },

  template: /*html*/`
<div class="data-picker">
  <div>
    activeWeather={{activeWeather}}
  </div>
  <ul>
    <li v-for="w in weatherDisplayOrder":key="w" 
      @click="$emit('weatherSelected', w)">
      <a href="#" :class="{ active: (activeWeather==w) }">
        {{ weatherList[w].text }} 
      </a> 
    </li>
  </ul>
</div>`
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

    function displayedJours() {
      //console.log( "displayedJours() typeof props.data.prevs is " + typeof props.data.prevs )
      const ret = []
      if (typeof props.data.prevs !== 'undefined') {
        for (var i = -1; i < 2; i++) {
          if (Object.hasOwn(props.data.prevs, i)) {
            ret.push(props.data.prevs[i])
          }
        }
      }
      return ret
    }

    return { displayedJours }
  },

  template: /*html*/`
<div class="maps-grid">
    <MapRowComponent
    v-for="(jour, idx) in displayedJours()"
    :key="idx"
    :prevsDuJour="jour"
    :data="data"
    :selections="selections"/>
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
 <div class="maps-row">
  <MapComponent v-for="(prev, idx) in prevsDuJour"
  :key="idx"
  :prev="prev"
  :data="data"
  :selections="selections"/>
</div>`
}


export const MapComponent = {
  props: {
    prev: Object,
    data: Object,
    selections: Object,
  },

  setup(props) {

    let _map_id = 0
    function mapId() {
      if (_map_id == 0) {
        _map_id = nextMapId()
      }
      return String(_map_id)
    }

    function mapTitle() {
      let weather = (typeof props.selections != null) ?
        weatherList[props.selections.activeWeather].text : ""
      let moment = (props.prev != null) ?
        props.prev.echeance : ""
      return moment + ' - ' + weather
    }

    // leaflet.Map object cannot be created in setup() 
    // because DOM element does not exist before onMounted()
    // we need a reference in component instance to control tooltips and displayed data
    let lMap = null
    let lBounds = null

    // keep references to markers for update/deletion
    let markers = []

    function initMap() {
      //  when timespan changes, components are cached/re-used by v-for algorithm
      // so just skip initMap because map and subzones do not change.
      // if (this.map) 
      //  return true;

      // format bounds in a leaflet-specific object
      let bbox = props.data.bbox
      lBounds = L.latLngBounds([[bbox.s, bbox.w], [bbox.n, bbox.e]])

      // setup main leaflet object
      lMap = L.map(mapId(), {
        center: lBounds.center,
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
        closePopupOnClick: true,
      })

      // add SVG map background
      let overlay = L.imageOverlay(svgPath(), lBounds)
      lMap.addLayer(overlay)
      lMap.setMaxBounds(lBounds)
      lMap.fitBounds(lBounds)
      lMap.setZoom(lMap.getBoundsZoom(lBounds, true))

      // trigger leaflet container resize on HTML element size change
      let elt = lMap.getContainer()
      let obs = new ResizeObserver((entries) => {
        lMap.setMaxBounds(lBounds)
        lMap.fitBounds(lBounds)
        lMap.setZoom(lMap.getBoundsZoom(lBounds, true))
        lMap.invalidateSize({ animate: false, pan: false })
      })
      obs.observe(elt)

      drawSubzones()

      // todo: add updated date in "attributions"

      // trigger markers on map creation
      // later activeWeather changes are handled with a watcher
      updateMarkers()
    }

    // update markers when activeWeather changes
    // use a getter ()=> to keep reactivity 
    // https://vuejs.org/guide/essentials/watchers.html#watch-source-types
    watch( ()=>props.selections.activeWeather, updateMarkers)


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
        pois.forEach(createMarker);

        // TODO
        // this.updateControl.setPrefix ("Màj : "+this.get_prevs.updated );
      }
    }

    function removeMarkers() {
      while (markers.length > 0) {
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

      if (prev.T !== null) {
        // donnée court-terme en priorité si disponibles
        mark.T = Math.round(prev.T)
        mark.txt = mark.T + "°"
      } else if (daily.T_min !== null && daily.T_max !== null) {
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

      let m = buildMarker(mark)

      // add a callback to make the marker clickable
      let target = 'http://www.' + poi.titre + '.zzzzzzz'
      m.on('click', ((e) => onMarkerClick(e, target)))

      // keep a reference for later cleanup and add to map
      markers.push(m)
      m.addTo(lMap)
    }


    function onMarkerClick(e, target) {
      console.log([target, e])
    }


    function buildMarker(mark) {
      if (mark.disabled) {
        return
      }
      //      console.log(["addMarker()", mark])
      let icon_width = 40
      if (props.selections.activeWeather == 'vent') {
        icon_width = 25    // icones du vent plus petites
      }

      // style pour le texte température min/max (qui contient un slash)
      let icon_text_style = ""
      /*
      if( ("string" == typeof e.icon_text || e.icon_text instanceof String) && 
              e.icon_text.indexOf("/") > -1 ) {
          (icon_text_style = "font-size: 12px;");  */

      let elt_a = '<a>' +
        '<img src="/pictos/' + mark.icon + '" ' +
        'alt="' + mark.desc + '" ' +
        'title="' + mark.titre + '" ' +
        'class="icon shape-weather" ' +
        'style="width: ' + icon_width + 'px"/>'

      if (mark.txt && "" != mark.txt && "NaN°" != mark.txt) {
        elt_a += '<span class="icon_text" style="' + icon_text_style + '">' +
          mark.txt + '</span>'
      }
      elt_a += "</a>"

      let mark_opts = {
        icon: L.divIcon({
          html: elt_a,
          className: "iconMap-1",
          iconSize: [icon_width, icon_width],
          iconAnchor: [icon_width / 2, icon_width / 2]
        })
      }

      // Inconsistent API order : GeoJSON=(lng,lat), Leaflet=(lat,lng)
      return L.marker([mark.coords[1], mark.coords[0]], mark_opts)
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
<div class="map-item">
  <div class="titre_carte"> {{ mapTitle() }} </div>
  <div :id="mapId()" class="map_component"></div>
</div>`

}



