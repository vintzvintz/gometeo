
import { ref,reactive, onMounted } from 'vue'

export const RootComponent = {

  setup() {

    // map data properties must be declared at component creation 
    // filled asynchronously by fetchMapdata() later
    const mapData = reactive({
      'name':'',
      'idtech':'',
      'taxonomy':'',
      'bbox': {},
      'subzones':new Array(),
      'prevs':{},
    })

    // selection of displayed data
    const selections = reactive( { 
      tooltipsEnabled: false,
      activeWeather: String("default")
      //activeTimespan: String("")
    })

    const breadcrumb = reactive([ 
      {name: "pays", path: "/path_france"},
      {name: "région", path: "/path_region" },
      {name: "dept", path: "/path_dept" },
    ])

    async function fetchMapdata() {
      console.log("enter fetchMapdata()")
      const res = await fetch( "/france/data")
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
    onMounted( fetchMapdata )

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

    //const moments = ref(['matin', 'après-midi', 'soirée', 'nuit' ])
    const displayedJours = (() => {
      console.log( 'in displayedJours() ')
      const ret = []
      for( let jour in props.data.prevs) {
        let n = parseInt(jour)
        if ( n<3 && n>=0) {
          ret.push( props.data.prevs[jour] )
        }
      }
      console.log( ret )
      return ret
    })

    return {displayedJours}
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

  props:{
    prevsDuJour: Array,
    data: Object,
    selections:Object,
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
    data : Object,
    selections : Object,
  },

  setup(props) {

    const weatherNames = new Map( [
      ["default", "Default"],
    ])

/*
      ["matin", "Matin"],
      ["après-midi", "Aprèm"],
      ["soirée", "Soir"],
      ["nuit", "Nuit"],
*/

    const mapId = function() {
      let t = Date.parse( props.prev.echeance )
      return String(t)
    }

    const mapTitle = function () {
      console.log( props.prev )
      console.log( props.data )
      console.log( props.selections)
      return weatherNames.get(props.selections.activeWeather) + ' - ' + props.prev.echeance
    }

    function initMap() {
      //  when timespan changes, components are cached/re-used by v-for algorithm
      // so just skip initMap because map and subzones do not change.
      // if (this.map) 
      //  return true;

      // setup Leaflet
      let bbox = props.data.bbox
      let bounds = L.latLngBounds([[bbox.s,bbox.w], [bbox.n,bbox.e]])
      let lMap = L.map( mapId(), {
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

      let overlay = L.imageOverlay(svgPath(), bounds);
      lMap.addLayer(overlay);
      lMap.setMaxBounds(bounds);
      lMap.fitBounds(bounds);
      lMap.setMinZoom(lMap.getBoundsZoom(bounds, true))
    }

    function svgPath() { 
      var img = new Image;
      img.src = '/france/svg';
      return img
    }

    onMounted( initMap )
    return {mapTitle, mapId}
  },


  template: /*html*/`
<div class="map_grid_item">
  <div class="titre_carte"> {{ mapTitle() }} </div>
  <div :id="mapId()" class="map_component"></div>
</div>`

}



