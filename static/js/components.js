import { ref, reactive, watch, computed, onMounted } from 'vue'

// id generator for mapComponents
let mapCount = 0
function nextMapId() {
  return ++mapCount
}

const weatherList = {
  "default": {
    text: "default",
    showTendance: false,
  },
  "prev": {
    text: "Prévisions",
    showTendance: true,
  },
  "vent": {
    text: "Vent",
    showTendance: false,
  },
  "ress": {
    text: "Ressenti",
    showTendance: false,
  },
  "humi": {
    text: "Humidité",
    showTendance: true,
  },
  "psea": {
    text: "Pression",
    showTendance: false,
  },
  "uv": {
    text: "UV",
    showTendance: true,
  }
}

const weatherDisplayOrder = [
  "prev", "vent", "ress", "humi", "psea", /*"uv",*/
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
      'chroniques': null,
    })

    // selection of displayed data
    const selections = reactive({
      tooltipsEnabled: true,
      activeWeather: "prev"
    })

    onMounted(() => {
      fetchMapdata()
      observeBodyWidth()
    })

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
    const tooltipsMinWidth = 800

    function onToggleTooltips() {
      selections.tooltipsEnabled = !selections.tooltipsEnabled   // reactive
    }
    function setTooltipsState(width) {
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

    <HighchartComponent 
    v-if="mapData.chroniques != null "
    :activeWeather="selections.activeWeather"
    :chroniques="mapData.chroniques"/>

  </section>
</header>
<!--    <h2 style="color: rgb(43, 70, 226);">2024-08-18 : Tests en cours ...<P></P> </h2> -->


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
 <!-- <a class="no-mobile" href="/about">A propos</a> -->
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


export const MapGridComponent = {

  props: {
    data: Object,
    selections: Object,
  },

  emits: ['setTooltipsState'],

  setup(props, ctx) {

    function displayedRows() {
      const firstDay = 0
      let showTendance = weatherList[props.selections.activeWeather].showTendance
      const ret = []
      if (typeof props.data.prevs !== 'undefined') {
        for (var i = firstDay; i < 14; i++) {
          if (!Object.hasOwn(props.data.prevs, i)) {
            continue
          }
          let row = props.data.prevs[i]
          if (row.long_terme && !showTendance) {
            continue
          }
          row["jour"] = i
          ret.push(row)
        }
      }
      return ret
    }

    return { displayedRows }
  },

  template: /*html*/`
<div class="maps-grid">
    <MapRowComponent
    v-for="(row, idx) in displayedRows()"
    :key="idx"
    :row="row"
    :data="data"
    :selections="selections"/>
</div>`
}


export const MapRowComponent = {

  props: {
    row: Object,
    data: Object,
    selections: Object,
  },

  setup(props) {

    function rowTitle() {
      let weather = (typeof props.selections != null) ?
        weatherList[props.selections.activeWeather].text : ""
      if (!props.row.long_terme) {
        return `${weather} J+${props.row.jour}`
      } else {
        return `Tendance J+${props.row.jour}`
      }
    }

    return { rowTitle }
  },

  template: /*html*/`
<div> {{ rowTitle() }} </div>
 <div class="maps-row">
  <MapComponent v-for="(prev, idx) in row.maps"
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
    // keep references to leaflet objects for user interactions
    let lMap = null
    let lBounds = null
    let lMarkers = []
    let lAttributionControl = null

    onMounted(() => {
      initMap()
      // update maps on selectors events
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


    function createMarker(poi, idx, all_pois) {
      const prev = poi.prev        // alias

      // prepare marker data for current poi
      const m = {
        title: poi.titre,
        coords: poi.coords,
        marker_width: 40,
        icon_width: 40,
        icon_text_style: "",   // avoid null checks in template
        txt: "",
        icon: prev.weather_icon,
        desc: prev.weather_description,
        Tmin: Math.round(prev.T_min),
        Tmax: Math.round(prev.T_max),
      }

      // représentation variable de la temperature selon prev.long_terme
      if (!prev.long_terme) {
        // donnée court-terme en priorité si disponibles
        m.txt = Math.round(prev.T) + '°'
      } else {
        m.txt = ` <span class="tmin">${m.Tmin}°</span>/<span class="tmax">${m.Tmax}°</span>`
        m.icon_text_style = 'font-size: 12px;'
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

      const kmPerHour = (mPerSecond) => (5 * Math.ceil(3.6 * mPerSecond / 5))

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

        m.txt = '<span>' + kmPerHour(prev.wind_speed) + '</span>'
        if (prev.wind_speed_gust >= 10) {
          m.txt += '<span style="color:red;">|' + kmPerHour(prev.wind_speed_gust) + '</span>'
        }

        // customisations pour les UV  
      } else if (w == "uv") {
        m.icon = "UV_" + prev.uv_index
        m.txt = "index uv " + prev.uv_index

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

        // customisation pour l'humidité relative
      } else if (w == "humi") {
        let hr_unit = "%";
        // représentation variable court-terme / long-terme
        if (!prev.long_terme) {
          m.txt = Math.round(prev.relative_humidity) + hr_unit
        } else {
          m.txt = `
          <span class="hr_min">
            ${Math.round(prev.relative_humidity_min)}${hr_unit}
          </span>/<span class="hr_max">
            ${Math.round(prev.relative_humidity_max)}${hr_unit}
          </span>`
          m.icon_text_style = "font-size: 12px;"
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
<div>
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
  <div class="tt_location">${m.title}</div>
  <img src="/pictos/${m.icon}" alt="${m.desc}" title="${m.desc}"/>
  <div class='tt_temp'>${m.txt}</div>
  <div class='tt_description'>${m.desc}</p>
  <div>
    Min : <span class='tmin'>${m.Tmin}°</span>
    Max : <span class='tmax'>${m.Tmax}°</span>
  </div>
</div>`
    }


    const updateDateOpts = Intl.DateTimeFormat("fr-FR", {
      day: "numeric",
      month: "short",
      hour: 'numeric',
      timeZone: "Europe/Paris"
    }).resolvedOptions()

    // display update time in "attribution" leaflet pre-defined control
    function showUpdateDate() {
      //let updated = new Date(props.prev.updated)
      let txt = "Màj : " +
        Intl.DateTimeFormat("fr-FR", updateDateOpts)
          .format(new Date(props.prev.updated))
      lAttributionControl.setPrefix(txt)
    }


    const titleDateOpts = Intl.DateTimeFormat("fr-FR", {
      weekday: "long",
      day: "numeric",
      month: "long",
      hour: 'numeric',
      timeZone: "Europe/Paris"
    }).resolvedOptions()

    function mapTitle() {
      if (!props.prev) {
        return "indisponible"
      }
      return Intl.DateTimeFormat("fr-FR", titleDateOpts)
        .format(new Date(props.prev.echeance))
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


    function optsTemperature() {
      return {
        title: 'Température',
        axeY1: '°C',
        series: {
          'T': { lineWidth: 1, color: '#0f1f0f', index: 50 },
          'Trange': {
            type: 'arearange',
            opacity: 0.1,
            index: 40,
            color: {
              linearGradient: {
                x1: 0,
                x2: 0,
                y1: 0,
                y2: 1
              },
              stops: [
                [0, '#FF5010'],
                [1, '#1060FF']
              ]
            }
          },
          // min-max range
          //'Tmax': { lineWidth: 1, color: '#D11' },
          //'Tmin': { type:"line", step:"right", lineWidth: 1, color: '#22D' },
        }
      }
    }

    function optsRessenti() {
      return {
        title: 'Température ressentie',
        axeY1: '°chill',
        series: {
          'Ress': { lineWidth: 1, color: '#444' },
        },
      }
    }

    function optsHumide() {
      return {
        title: 'Humidité relative',
        axeY1: '%',
        series: {
          //          'Hrel': { lineWidth: 1, color: '#BBB' },
          //          'Hmax': { lineWidth: 1, color: '#1D1' },
          //          'Hmin': { lineWidth: 1, color: '#DD1' },
          'Hrel': { lineWidth: 1, color: '#0f1f0f', index: 50 },
          'Hrange': {
            type: 'arearange',
            opacity: 0.1,
            index: 40,
            color: {
              linearGradient: {
                x1: 0,
                x2: 0,
                y1: 0,
                y2: 1
              },
              stops: [
                [0, '#1D1'],
                [1, '#FD3']
              ]
            }
          },
        },
      }
    }
    function optsPression() {
      return {
        title: 'Pression au niveau de la mer',
        axeY1: 'hPa',
        series: {
          'Psea': { lineWidth: 1, color: '#222' },
        },
      }
    }

    function optsCloudCover() {
      return {
        title: 'Couverture nuageuse',
        axeY1: '%',
        series: {
          'Cloud': { lineWidth: 1, color: '#222' },
        },
      }
    }


    const hcConf = computed(() => {
      let w = props.activeWeather
      if (w == "prev" || w == "vent" || w == "uv") {
        return optsTemperature()
      } else if (w == "ress") {
        return optsRessenti()
      } else if (w == "humi") {
        return optsHumide()
      } else if (w == "psea") {
        return optsPression()
      } else if (w == "cloud") {
        return optsCloudCover()
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
        accessibility: {
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
        tootip: { enabled: true },
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

    function hcReset(hc, graphTitle, yTitle) {
      // remove all previous series and Y-axis
      while (hc.series.length) {
        hc.series[0].remove(false)
      }
      hc.axes.forEach((axe) => {
        axe.isXAxis || axe.remove(false)
      })

      hc.setTitle(
        { text: graphTitle },
        {},   // no subtitle
        false // no redraw
      )
      hc.addAxis(
        {
          id: 'axeY1',
          title: {
            text: yTitle,
            align: 'high',
            offset: 0,
            rotation: 0,
            y: -20,
          },
        },
        false, // not an Xaxis
        false, // no redraw
      )
    }

    function updateGraph() {
      let conf = hcConf.value   // computed property

      hcReset(hcObj, conf.title, conf.axeY1)

      // iterate over configured series for current activeWeather
      for (let sName in conf.series) {
        // chroniques is a set of chroniques of the same "type" (T, Hrel, Twindchill, etc...)
        // a chronique is an array of [ ts, val ] pairs ( one per POI)
        let chroniques = props.chroniques[sName]
        for (let chronique of chroniques) {
          // deepcopy intended - graph options are not shared between each individual serie
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



