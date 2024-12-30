
import { ref,reactive, onMounted } from 'vue'

export const RootComponent = {

  setup() {


    const mapData = reactive({})


    const tooltipsEnabled = ref( false )
    const activeWeather = ref("")
    const activeTimespan = ref("")

    const breadcrumb = reactive([ 
      {name: "pays", path: "/path_france"},
      {name: "région", path: "/path_region" },
      {name: "dept", path: "/path_dept" },
    ])

//    const prevs = reactive(new Map())

    async function fetchMapdata() {
      const res = await fetch( "/france/data")
      const data = await res.json() 

      mapData.name = data.name
      mapData.idtech = data.idtech
      mapData.taxonomy = data.taxonomy
      mapData.bbox = data.bbox
      mapData.subzones = data.subzones
      mapData.prevs = data.prevs
    }

    // get Prevs
    onMounted( fetchMapdata )

    // only returned items are available in template
    return {
      mapData,
    }
  },


  template:/*html*/`
  <p>Wesh root</p>
  <p>mapname={{mapData.name}}</p>
  <p>bbox={{mapData.bbox}}</p>
  <p>subzones={{mapData.subzones}}</p>
  <p>prevs={{mapData.prevs}}</p>
  `,

  templatexxx: /*html*/ `
  <header>
  <Breadcrumb :breadcrumb="breadcrumb"/>
  <section class="selecteurs">
  <p>mapname={{mapname}}</p>
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
   :bbox='bbox'
   :subzones='subzones'
   :prevs='prevs'/>
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
    activeweather : String,
  },

  setup(props) {

    //const moments = ref(['matin', 'après-midi', 'soirée', 'nuit' ])
    const displayedJours = (() => {
/*      const ret = []
      for( let jour in props.prevs) {
        let n = parseInt(jour)
        if ( n<5 && n>=0) {
          ret.push( jour )
        }
      }
      return ret
      */
     console.log( 'in displayedJours()' )
    return props.prevs.length
    })

    return {displayedJours}
  },

  template: /*html*/`
<div class="map_grid_container">
  <p>MapGrid component displayedJours={{displayedJours()}} </p> <p>prevs.length={{prevs.length}}</p>
  <!--    <map-component v-for='rang in get_rangs' v-bind:key="rang" v-bind:rang="rang">Chargement...</map-component> -->
  <div v-for="jour in displayedJours" :key="jour">  

    <MapComponent v-for="(prev, idx) in prevs[jour]"
      :key="idx"
      :prev="prev"
      :activeweather="activeweather">
    </MapComponent>
  </div>
</div>`
}

export const MapComponent = {
  props: {
    prev : Object,
    activeWeather : String,
  },

  template: /*html*/`
  <p>MapComponent activeWeather={{activeWeather}}</p>
  <p>updated {{prev.Updated}} - échéance {{prev.Time}}</p>
  `
}


