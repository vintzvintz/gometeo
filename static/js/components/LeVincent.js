
import { ref } from 'vue'

export const RootComponent = {

  template: /*html*/ `
  <header>
  <Breadcrumb/>
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
  <div class="map_grid_container">
<!--    <map-component v-for='rang in get_rangs' v-bind:key="rang" v-bind:rang="rang">Chargement...</map-component> -->
  <MapGridComponent/>
  </div>
</main>`

}

export const Breadcrumb = {
  props: {
    breadcrumb: String
  },

  template: /*html*/`
<nav class="topnav">
  Navigation : {{breadcrumb}}
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
<!--    <map-component v-for='rang in get_rangs' v-bind:key="rang" v-bind:rang="rang">Chargement...</map-component> -->
  <p>MapGrid component</p>
  <MapComponent title="Carte 1"/>
  <MapComponent title="Carte 2"/>
  <MapComponent title="Carte 3"/>`
}

export const MapComponent = {

  props: {
    title: String,
  },
  template: /*html*/`
  <p>MapComponent {{title}}</p>`
}
