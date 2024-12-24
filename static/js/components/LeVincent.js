
import { ref,reactive } from 'vue'

export const RootComponent = {

  setup: function()  {

    const mapname = ref("default map name")
    const idtech = ref("")
    const taxonomy = ref("")
    const bbox = reactive({})

    const subzones = reactive([])

    const breadcrumb = reactive([ 
      {name: "pays", path: "/path_france"},
      {name: "région", path: "/path_region" },
      {name: "dept", path: "/path_dept" },
    ])

    const prevs = reactive({})

    const tooltipsEnabled = ref( false )
    const activeWeather = ref("")
    const activeTimespan = ref("")

    async function fetchPrevs() {
      const res = await fetch( "/france/data")
      const data = await res.json() 
      console.log(data)
      
      mapname.value = data.name
      idtech.value = data.idtech
      taxonomy.value = data.taxonomy
      bbox.value = data.bbox
      subzones.value = data.subzones
      prevs.value = data.Prevs
    }

    // get Prevs
    fetchPrevs()
    console.log("FetchPrev() returned")

    // only returned items are available in template
    return {
      breadcrumb,
    }
  },


  template: /*html*/ `
  <header>
  <Breadcrumb :breadcrumb="breadcrumb"/>
  <section class="selecteurs">
    <DataPicker/>
  <div>
    <TimespanPicker/>
    <TooltipsToggler/>
  </div>
<!--  <highchart-graph v-if="display_graph"></highchart-graph> -->
  </section>
</header>
<!--    <h2 style="color: rgb(43, 70, 226);">2024-08-18 : Tests en cours ...<P></P> </h2> -->
<main class="content">
  <MapGridComponent/>
</main>`

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
  template: /*html*/`
<div class="map_grid_container">
  <p>MapGrid component</p>
  <!--    <map-component v-for='rang in get_rangs' v-bind:key="rang" v-bind:rang="rang">Chargement...</map-component> -->
  <MapComponent title="Carte 1"/>
  <MapComponent title="Carte 2"/>
  <MapComponent title="Carte 3"/>
</div>`
}

export const MapComponent = {

  props: {
    title: String,
  },
  template: /*html*/`
  <p>MapComponent {{title}}</p>`
}
