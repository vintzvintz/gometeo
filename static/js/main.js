
import { createApp, ref } from 'vue'

import {
  RootComponent, 
  Breadcrumb, 
  WeatherPicker,
  TimespanPicker,
  TooltipsToggler,
  MapGridComponent,
  MapRowComponent,
  MapComponent,
} from './components/LeVincent.js'

const app = createApp({
  setup() {
    const message = ref("Le createApps message")
    return { message }
  },

  template: /*html*/`<RootComponent/>`
})

app.component("RootComponent", RootComponent)
app.component("Breadcrumb", Breadcrumb)
app.component("WeatherPicker", WeatherPicker)
app.component("TimespanPicker", TimespanPicker)
app.component("TooltipsToggler", TooltipsToggler)
app.component("MapGridComponent", MapGridComponent)
app.component("MapRowComponent", MapRowComponent)
app.component("MapComponent", MapComponent)

app.mount('#vuejs_root')
