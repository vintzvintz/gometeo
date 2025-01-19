import { ref, reactive, watch, computed, onMounted } from 'vue'

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

    onMounted(() => {
      fetchMapdata()
      observeBodyWidth()
    })

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
      'chroniques': null,
    })

    // selection of displayed data
    const selections = reactive({
      tooltipsEnabled: true,
      activeWeather: "prev"
      //activeTimespan: String("")
    })

/*    watch(() => selections.tooltipsEnabled, () => {
      console.log("tooltipsEnabled=" + selections.tooltipsEnabled)
    })*/

    async function fetchMapdata() {
      console.log(`fetchMapdata() path=${props.path}`)
      const res = await fetch(`/${props.path}/data`)
      const data = await res.json()

      // cant replace whole mapData (reactive) object because it is reactive,
      // so we update each property explicitly
      mapData.name = data.name
      mapData.path = data.path
      mapData.breadcrumb = data.breadcrumb
      mapData.idtech = data.idtech
      mapData.taxonomy = data.taxonomy
      mapData.bbox = data.bbox
      mapData.subzones = data.subzones
      mapData.prevs = data.prevs
      mapData.chroniques = data.chroniques
      console.log("fetchMapdata() exit")
    }

    // callback when WeatherPicker emits a 'weatherSelected' event
    function onWeatherSelected(id) {
      if (id != "uv") {
        selections.activeWeather = id   // reactive
      }
    }

    // tooltips visibility
    const tooltipsMinWidth = 600

    function onToggleTooltips() {
      selections.tooltipsEnabled = !selections.tooltipsEnabled   // reactive
    }
    function setTooltipsState(width) {
      console.log("setTooltipsState() width", width)
      selections.tooltipsEnabled = tooltipsMinWidth < width // reactive
    }

      // react to body.resize events
      function observeBodyWidth() {
      const obs = new ResizeObserver((entries) => {
        for (let entry of entries) {
          if (entry.contentBoxSize) {
            setTooltipsState(entry.contentBoxSize[0].inlineSize)
          }
        }
      })
      obs.observe(document.body)
    }

    // returned objects are available in template
    return {
      mapData,
      selections,
      onWeatherSelected,
      onToggleTooltips,
      setTooltipsState,
    }
  },

  template: /*html*/ `
  <header>
  
  <TopNav 
  :breadcrumb="mapData.breadcrumb"
  :tooltipsEnabled="selections.tooltipsEnabled"
  @toggleTooltips="onToggleTooltips"/>

  <section class="selecteurs">
    <WeatherPicker 
    :activeWeather="selections.activeWeather"
    @weatherSelected="onWeatherSelected" />

<!--    <TooltipsToggler
    :tooltipsEnabled="selections.tooltipsEnabled"
    @toggleTooltips="onToggleTooltips"/> -->

    <HighchartComponent 
    v-if="mapData.chroniques != null "
    :activeWeather="selections.activeWeather"
    :chroniques="mapData.chroniques"/>

  </section>
</header>
<!--    <h2 style="color: rgb(43, 70, 226);">2024-08-18 : Tests en cours ...<P></P> </h2> -->

<!--   v-if="mapData.path!=null" -->
<main class="content">
  <MapGridComponent
  :selections="selections"
  :data="mapData" 
  @setTooltipsState="setTooltipsState"/>
</main>

<!--<footer class="footer"> <p>Footer</p> </footer> -->
`
}

export const TopNav = {
  props: {
    breadcrumb: Array,
    tooltipsEnabled: Boolean,
  },

  emits: ['toggleTooltips'],

  setup(props) {
  },

  template: /*html*/`
<nav class="topnav">
  <a v-for="item in breadcrumb" :href="item.path">{{item.nom}}</a>
  <div class="spacer"></div>
  <a class="no-mobile" href="/about">A propos</a>
  <a class="no-mobile" @click="$emit('toggleTooltips')" > 
     Tooltips : {{tooltipsEnabled ? "Oui" : "Non"}}
  </a>
</nav>`
}

export const WeatherPicker = {

  emits: ['weatherSelected'],

  props: {
    activeWeather: String,
  },

  setup(props) {
    // make module-level globals available in template
    return { weatherList, weatherDisplayOrder }
  },

  template: /*html*/`
<div class="data-picker">
  <a v-for="w in weatherDisplayOrder" 
  :key="w"
  :class="{ active: (activeWeather==w) }"
  @click="$emit('weatherSelected', w)" >
      {{ weatherList[w].text }} 
  </a> 
</div>`
}
/*
export const TooltipsToggler = {

  emits: ['toggleTooltips'],

  props: {
    tooltipsEnabled: Boolean,
  },

  setup(props) {
  },

  template: `
<div id="tooltip_toggler"@click="$emit('toggleTooltips')">
  <a :class="{active:tooltipsEnabled}" href="#">
  Tooltips : {{tooltipsEnabled ? "Oui" : "Non"}}
  </a>
</div>`
}
*/
/*
export const TimespanPicker = {

  template: `
  <p>TimespanPicker component</p>`
}
*/

export const MapGridComponent = {

  props: {
    data: Object,
    selections: Object,
  },

  emits: ['setTooltipsState'],

  setup(props, ctx) {

    function displayedJours() {
      const ret = []
      if (typeof props.data.prevs !== 'undefined') {
        for (var i = -1; i < 4; i++) {
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

    // leaflet.Map object cannot be created before onMounted()
    // because DOM container element does not exist yet 
    // keep references to leaflet objects for update/deletion upon user interaction
    let lMap = null
    let lBounds = null
    let lMarkers = []
    let lAttributionControl = null
    onMounted(() => {
      initMap()

      // update maps on selectors change
      // use a getter ()=> to keep reactivity  
      // cf. https://vuejs.org/guide/essentials/watchers.html#watch-source-types
      watch(() => props.selections.activeWeather, updateMarkers)
      watch(() => props.selections.tooltipsEnabled, updateTooltipsVisibility)
    })

    // mapId is defined at component creation from a module-level global var.
    let _map_id = 0
    function mapId() {
      if (_map_id == 0) {
        _map_id = nextMapId()
      }
      return String(_map_id)
    }


    function initMap() {

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
      let svgElt = new Image
      svgElt.src = `/${props.data.path}/svg`
      lMap.addLayer(L.imageOverlay(svgElt, lBounds))

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

      addSubzones()

      // trigger markers and resize once on map creation
      // next activeWeather changes are handled with a watcher and reactivity
      resizeMap()
      updateMarkers()
      updateTooltipsVisibility()
    }


    function addSubzones() {
      if (props.data.subzones === null) {
        return
      }
      const szPane = lMap.createPane('subzones')
      props.data.subzones.forEach((sz) => {
        let path = sz.properties.customPath
        let nom = sz.properties.prop0.nom
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
        // remove previous markers before recreating new ones
        while (lMarkers.length > 0) {
          lMap.removeLayer(lMarkers.pop())
        }
        pois.forEach(createMarker)
        showUpdateDate()
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
        //disabled: false,
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
      let yOffset = 0
      if (poi.coords[1] < (props.data.bbox.n + props.data.bbox.s) / 2) {
        m.tt_direction = 'top'
        yOffset = -30
      }
      else {
        m.tt_direction = 'bottom'
        yOffset = 10
      }
      m.tt_offset = (poi.coords[0] < (props.data.bbox.w + props.data.bbox.e) / 2) ?
        L.point(100, yOffset) : L.point(-100, yOffset)

      const msToKmh = (mPerSecond) => (5 * Math.ceil(3.6 * mPerSecond / 5))

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

      let marker = markerTemplate(m)

      let tt_html = tooltipTemplate(m)
      marker.bindTooltip(tt_html, {
        sticky: false,
        direction: m.tt_direction,
        offset: m.tt_offset,
      })

      // attach a callback to make the marker clickable
      // TODO : trouver la subzone contenant le marker ( a faire plutot server-side ?) 
      //let target = 'http://www.' + poi.titre + '.zzzzzzz'
      //marker.on('click', ((e) => console.log([target, e]))
      marker.addTo(lMap)

      // keep a reference for later cleanup
      lMarkers.push(marker)
    }


    function markerTemplate(m) {
      let elt_a = /*html*/`
<div class="div-icon">
  <img src="/pictos/${m.icon}" 
       alt="${m.desc}"
       title="${m.title}"
       style="width: ${m.icon_width}px"/>`

      if ("" != m.txt && "NaN°" != m.txt) {
        elt_a += `<div class="icon-text" style="${m.icon_text_style}">
          ${m.txt}</div>`
      }
      elt_a += /*html*/`</div>`

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

    function tooltipTemplate(m) {
      return /*html*/`
<div class="map_tooltip">
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

    // display update time in "attribution" leaflet pre-defined control
    function showUpdateDate() {
      //let updated = new Date(props.prev.updated)
      let txt = "Màj : " +
        Intl.DateTimeFormat("fr-FR", dateFormatOpts)
          .format(new Date(props.prev.updated))
      lAttributionControl.setPrefix(txt)
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


    function updateTooltipsVisibility() {
      lMap && lMap.getPane('markerPane') &&
        lMap.getPane('markerPane').childNodes.forEach(function (m) {
          let classes = m.classList
          props.selections.tooltipsEnabled ?
            classes.add('leaflet-interactive') :
            classes.remove('leaflet-interactive')
        })
    }

    return { mapTitle, mapId }
  },

  template: /*html*/`
<div class="map-item">
  <div class="titre_carte"> {{ mapTitle() }} </div>
  <div :id="mapId()" class="map_component"></div>
</div>`

}

export const HighchartComponent = {
  props: {
    chroniques: Object,
    activeWeather: String,
  },

  setup(props) {

    onMounted(() => {
      initGraph()
      updateGraph()
      watch(() => props.activeWeather, updateGraph)
    })

    let hcObj = null   // highchart object created in initGraph()

    const hcConf = computed(() => {
      let w = props.activeWeather
      if (w == "prev" || w == "vent" || w == "uv") {
        return {
          title: 'Temperature',
          axeY1: '°C',
          series: {
            'T': { lineWidth: 1, color: '#BBB' },
            'Tmax': { lineWidth: 1, color: '#D11' },
            'Tmin': { lineWidth: 1, color: '#22D' },
          },
        }
      } else if (w == "ress") {
        return {
          title: 'Température ressentie',
          axeY1: 'indice de refroidissement',
          series: {
            'Ress': { lineWidth: 1, color: '#444' },
          },
        }
      } else if (w == "humi") {
        return {
          title: 'Humidité relative',
          axeY1: '%',
          series: {
            'Hrel': { lineWidth: 1, color: '#BBB' },
            'Hmax': { lineWidth: 1, color: '#1D1' },
            'Hmin': { lineWidth: 1, color: '#DD1' },
          },
        }
      } else if (w == "psea") {
        return {
          title: 'Pression au niveau de la mer',
          axeY1: 'hPa',
          series: {
            'Psea': { lineWidth: 1, color: '#222' },
          },
        }
      } else if (w == "cloud") {
        return {
          title: 'Couverture nuageuse',
          axeY1: '%',
          series: {
            'Cloud': { lineWidth: 1, color: '#222' },
          },
        }
      } else {  // default
        return {
          title: '',
          axeY1: '',
          series: {},
        }
      }
    })

    // mounts a highchart object on the template <div> container
    function initGraph() {
      hcObj = Highcharts.chart("highchartContainer", {
        chart: {
          type: 'spline',
        },
        accessibility:{
          enabled: false,
        },
        plotOptions: {
          series: {
            color: '#666699',
            enableMouseTracking: false,
            marker: { enabled: false },
            showInLegend: false,
            states: { hover: { enabled: false } }
          },
          spline: { lineWidth: 1, turboThresold: 3 }
        },
        tootip: { enabled: false },
        xAxis: {
          id: 'axeX',
          type: 'datetime',
          dateTimeLabelFormats: { day: '%e %b' },
          floor: Date.now() - 24 * 3600 * 1000,
          plotLines: [{
            color: '#FC0FC5',
            value: Date.now(),
            width: 2,
            zIndex: 12,
          }],
        },
      })
    }

    function updateGraph() {
      let conf = hcConf.value   // computed property
      // remove all previous series and Y-axis
      while (hcObj.series.length) {
        hcObj.series[0].remove(false)
      }
      hcObj.axes.forEach((axe) => {
        axe.isXAxis || axe.remove(false)
      })

      hcObj.setTitle(
        { text: conf.title },
        {},   // no subtitle
        false // no redraw
      )
      hcObj.addAxis(
        { id: 'axeY1', title: { text: conf.axeY1 } },
        false, // no Xaxis
        false // no redraw
      )

      // iterate over configured series for current activeWeather
      for (let sName in conf.series) {
        // chroniques is a set of chroniques of the same "type" (T, Hrel, Twindchill, etc...)
        // a chronique is an array of [ ts, val ] pairs ( one per POI)
        let chroniques = props.chroniques[sName]
        for (let chronique of chroniques) {
          // deepcopy intended - draw options are not shared anmong each individual serie
          let opts = Object.assign({}, conf.series[sName]);
          opts.data = chronique
          hcObj.addSeries(
            opts,
            false,  // do not redraw after each serie
          )
        }
      }
      hcObj.redraw()
    }

    return {}
  },

  template: /*html*/`
<div id="highchartContainer"></div>`
}



