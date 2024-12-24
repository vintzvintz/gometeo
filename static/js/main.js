
import { createApp, ref } from 'vue'

import {
  RootComponent, 
  Breadcrumb, 
  DataPicker,
  TimespanPicker,
  TooltipsToggler,
  MapGridComponent,
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
app.component("DataPicker", DataPicker)
app.component("TimespanPicker", TimespanPicker)
app.component("TooltipsToggler", TooltipsToggler)
app.component("MapGridComponent", MapGridComponent)
app.component("MapComponent", MapComponent)

app.mount('#vuejs_root')
