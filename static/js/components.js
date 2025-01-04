import { ref, reactive, watch, onMounted } from 'vue'

// id generator for mapComponents
let mapCount = 0
function nextMapId() {
  return ++mapCount
}

let dateFormatOpts = Intl.DateTimeFormat("fr-FR", {
  day: "numeric",
  month: "long",
  hour: 'numeric',
  timeZone: "Europe/Paris"
}).resolvedOptions()


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
  "uv": {
    text: "UV",
  }
}

const weatherDisplayOrder = [
  "prev", "vent", "ress", "humi", "psea", "uv",
]


export const RootComponent = {

  props: {
    path: String
  },

  setup(props) {

    // map data properties must be declared at component creation 
    // filled asynchronously by fetchMapdata() later
    const mapData = reactive({
      'path': null,
      'name': null,
      'idtech': null,
      'taxonomy': null,
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
      console.log(`fetchMapdata() path=${props.path}`)
      const res = await fetch(`/${props.path}/data`)
      const data = await res.json()

      mapData.name = data.name
      mapData.path = data.path
      mapData.idtech = data.idtech
      mapData.taxonomy = data.taxonomy
      mapData.bbox = data.bbox
      mapData.subzones = data.subzones
      mapData.prevs = data.prevs
      console.log("fetchMapdata() exit")
    }

    // callback when WeatherPicker emits a 'weatherSelected' event
    function onWeatherSelected(id) {
      selections.activeWeather = id   // reactive
    }

    // get Prevs when static page is loaded
    onMounted(() => {
      console.log(`onMounted() path=${props.path}`)
      fetchMapdata()

    })

    // only returned items are available in template
    return {
      mapData, selections, onWeatherSelected, breadcrumb
    }
  },

  template: /*html*/ `
  <header>
  <Breadcrumb :breadcrumb="breadcrumb"/>
  <section class="selecteurs">
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
        for (var i = -1; i < 12; i++) {
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
      if (!props.prev) {
        return "indisponible"
      }
      let moment = Intl.DateTimeFormat("fr-FR", dateFormatOpts)
        .format(new Date(props.prev.echeance))
      let weather = (typeof props.selections != null) ?
        weatherList[props.selections.activeWeather].text : ""
      return `${weather} ${moment}`
    }

    // leaflet.Map object cannot be created in setup() 
    // because DOM element does not exist before onMounted()
    // we need a reference in component instance to control tooltips and displayed data
    let lMap = null
    let lBounds = null

    // keep references to markers for update/deletion
    let lMarkers = []
    let lAttributionControl = null

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

      // add update info
      lAttributionControl = L.control
        .attribution({ prefix: "" })
        .addTo(lMap)

      function resizeMap() {
        lMap.setMaxBounds(lBounds)
        lMap.fitBounds(lBounds)
        lMap.setZoom(lMap.getBoundsZoom(lBounds, true))
      }

      // trigger leaflet container resize on HTML element size change
      let elt = lMap.getContainer()
      let obs = new ResizeObserver((entries) => {
        resizeMap()
        lMap.invalidateSize({ animate: false, pan: false })
      })
      obs.observe(elt)

      drawSubzones()
      resizeMap()

      // trigger markers once on map creation
      // next activeWeather changes are handled with a watcher
      updateMarkers()
    }

    // update markers when activeWeather changes
    // use a getter ()=> to keep reactivity 
    // https://vuejs.org/guide/essentials/watchers.html#watch-source-types
    watch(() => props.selections.activeWeather, updateMarkers)


    // display update time in "attribution" leaflet pre-defined control
    function showUpdateDate() {
      //let updated = new Date(props.prev.updated)
      let txt = "Màj : " + 
        Intl.DateTimeFormat("fr-FR", dateFormatOpts)
        .format(new Date(props.prev.updated))
      lAttributionControl.setPrefix(txt)
    }

    function svgPath() {
      var img = new Image
      img.src = `/${props.data.path}/svg`
      return img
    }

    function drawSubzones() {
      if (props.data.subzones === null) {
        return
      }

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
        showUpdateDate()
      }
    }

    function removeMarkers() {
      while (lMarkers.length > 0) {
        lMap.removeLayer(lMarkers.pop());
      }
    }

    function createMarker(poi, idx, all_prevs) {

      // local aliases
      const prev = poi.prev
      const daily = poi.daily

      // accumulate marker data for current poi
      const m = {
        title: poi.titre,
        coords: poi.coords,
        marker_width: 40,
        icon_width: 40,
        icon_text_style: "",   // avoid null checks in template
        txt: "",
        disabled: false,
        icon: prev.weather_icon,
        desc: prev.weather_description,
        Tmin: Math.round(daily.T_min),
        Tmax: Math.round(daily.T_max),
      }

      // TODO: fallback on daily_weather_desc if weather_desc == null
      // TODO: fallback on daily_weather_icon if prev.weather_icon == null

      // 3 représentations possibles pour la temperature :
      // - prev.T si disponible
      // - sinon daily.tmin/tmax
      // - sinon pas de marker
      if (prev.T !== null) {
        // donnée court-terme en priorité si disponibles
        m.txt = Math.round(prev.T) + '°'
      } else if (daily.T_min !== null && daily.T_max !== null) {
        // donnée long-terme (dailies) 
        m.txt = ` <span class="tmin">${m.Tmin}°</span>/<span class="tmax">${m.Tmax}°</span>`
        m.icon_text_style = 'font-size: 12px;'
      } else {
        // pas de marker si temperature indisponible
        return
      }

      // tooltip position
      m.tt_direction = (poi.coords[1] < (props.data.bbox.n + props.data.bbox.s) / 2) ?
        'top' : 'bottom'
      m.tt_offset = (poi.coords[0] < (props.data.bbox.w + props.data.bbox.e) / 2) ?
        L.point(100, 10) : L.point(-100, 10)

      // other customizations depending on activeWeather
      let w = props.selections.activeWeather

      // customisations pour le vent
      if (w == "vent") {
        // marker disabled if wind data is not available
        if (prev.wind_speed == null) {
          return
        }
        m.icon = prev.wind_icon;
        if (-1 == m.icon) {
          m.icon = "Variable";
        }
        m.icon_width = 25

        m.txt = '<span>' + msToKmh(prev.wind_speed) + '</span>'
        if (prev.wind_speed_gust >= 10) {
          m.txt += '<span style="color:red;">|' + msToKmh(prev.wind_speed_gust) + '</span>'
        }

        // customisations pour les UV  
      } else if (w == "uv") {
        if (daily.uv_index == null) {
          return
        }
        m.icon = "UV_" + prev.uv_index
        m.txt = ""

        // customisations pour la temp ressentie
      } else if (w == "ress") {
        if (prev.T_windchill == null) {
          return
        }
        m.txt = '<span style="color: brown;">' +
          Math.round(parseFloat(prev.T_windchill)) + '</span>'

        // customisations pour la pression au niveau de la mer
      } else if (w == "psea") {
        if (prev.P_sea == null) {
          return
        }
        m.txt = String(Math.round(parseFloat(prev.P_sea)))

        // customisations pour la couverture nuageuse
      } else if (w == "cloud") {
        if (prev.total_cloud_cover == null) {
          return
        }
        m.txt = prev.total_cloud_cover + "%"

        // add an icon_text_style "font-size: 12px;" on min/max values
      } else if (w == "humi") {

        let hr_unit = "%";
        if (prev.relative_humidity !== null) {
          // court terme
          m.txt = Math.round(prev.relative_humidity) + hr_unit
        } else if (prev.relative_humidity_min !== null) {
          // a long terme
          m.txt = `
          <span class="hr_min">
            ${Math.round(prev.relative_humidity_min)}${hr_unit}
          </span>/<span class="hr_max">
            ${Math.round(prev.relative_humidity_max)}${hr_unit}
          </span>`
          m.icon_text_style = "font-size: 12px;"
        } else {
          return
        }

      } else if (w == "prev") {
        // placeholder for future prev-specific stuff
      } else {
        console.log('activeWeather value: ' + w)
        return
      }

      let marker = buildMarker(m)

      let tt_html = buildTooltip(m)
      marker.bindTooltip(tt_html, {
        sticky: false,
        direction: m.tt_direction,
        offset: m.tt_offset,
      })


      // attach a callback to make the marker clickable
      let target = 'http://www.' + poi.titre + '.zzzzzzz'
      marker.on('click', ((e) => onMarkerClick(e, target)))
        .addTo(lMap)

      // keep a reference for later cleanup
      lMarkers.push(marker)
    }

    function onMarkerClick(e, target) {
      console.log([target, e])
    }

    function msToKmh(mPerSecond) {
      return 5 * Math.ceil(3.6 * mPerSecond / 5)
    }

    function buildMarker(m) {
      let elt_a = `<div class="div-icon">
  <img src="/pictos/${m.icon}" 
       alt="${m.desc}"
       title="${m.title}"
       style="width: ${m.icon_width}px"/>`

      if ("" != m.txt && "NaN°" != m.txt) {
        elt_a += `<div class="icon-text" style="${m.icon_text_style}">
          ${m.txt}</div>`
      }
      elt_a += `</div>`

      let mark_opts = {
        icon: L.divIcon({
          html: elt_a,
          className: "divIcon",
          //  iconSize: [m.icon_width, m.icon_width],
          iconAnchor: [m.icon_width / 2, m.icon_width / 2]
        })
      }

      // Inconsistent API order : GeoJSON=(lng,lat), Leaflet=(lat,lng)
      return L.marker([m.coords[1], m.coords[0]], mark_opts)
    }

    function buildTooltip(m) {
      return `<div class="map_tooltip">
  <h3 class="map_tooltip_location">${m.title}</h3>
  <img src="/pictos/${m.icon}" 
        alt="${m.desc}"
        title="${m.desc}"
        style="width: ${m.icon_width}px"/>
  <div class='map_tooltip_temp'>${m.txt}</div>
  <p class='map_tooltip_description'>${m.desc}</p>
  <p>
    Min : <span class='temp-min'>${m.Tmin}°</span>
    Max : <span class='temp-max'>${m.Tmax}°</span>
  </p>
</div>`
    }

    onMounted(initMap)

    return { mapTitle, mapId }
  },

  template: /*html*/`
<div class="map-item">
  <div class="titre_carte"> {{ mapTitle() }} </div>
  <div :id="mapId()" class="map_component"></div>
</div>`

}



