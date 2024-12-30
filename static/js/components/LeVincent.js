
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
    const tooltipsEnabled = ref( false )
    const activeWeather = ref("default_activeWeather")
    //const activeTimespan = ref("")

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
      mapData, activeWeather, breadcrumb
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
   :bbox='mapData.bbox'
   :subzones='mapData.subzones'
   :prevs='mapData.prevs'
   :activeWeather='activeWeather'
   />
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
    prevs: Object,
    bbox: Object,
    subzones:Array,
    activeWeather : String,
  },

  setup(props) {

    //const moments = ref(['matin', 'après-midi', 'soirée', 'nuit' ])
    const displayedJours = (() => {
      console.log( 'in displayedJours() ')
      const ret = []
      for( let jour in props.prevs) {
        let n = parseInt(jour)
        if ( n<3 && n>=0) {
          ret.push( props.prevs[jour] )
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
    :activeWeather="activeWeather"
    />
  </div>
</div>`
}


export const MapRowComponent = {

  props:{
    prevsDuJour: Array,
    activeWeather: String,
  },

  setup(props) {
  },

  template: /*html*/`
  <p> ... inside MapRowComponent ... </p>
  <MapComponent v-for="(prev, idx) in prevsDuJour"
  :key="idx"
  :prev="prev"
  :activeWeather="activeWeather">
</MapComponent>`
}


export const MapComponent = {
  props: {
    prev : Object,
    activeWeather : String,
  },

  setup(props) {

    const getWeatherTitle = function () { 
      return "title"
    }

    const getEcheance = (() => {
      return "echeance"
    })
    return {getWeatherTitle, getEcheance}
  },

  template: /*html*/`
<div class="map_grid_item"><div class="titre_carte">
  {{getWeatherTitle()}} - {{getEcheance()}}
</div></div>`

}



