import { createApp, ref } from 'vue'

import {
  RootComponent,
  TopNav,
  WeatherPicker,
  // TimespanPicker,
  // TooltipsToggler,
  MapGridComponent,
  MapRowComponent,
  MapComponent,
  HighchartComponent,
} from 'components'


export function createMeteoApp(mountElt, path, cacheId) {

  const app = createApp({
    props: {
      path: String,
      cacheId: String,
    },
    setup() {
    },
    template: /*html*/`<RootComponent :path="path" :cacheId="cacheId"/>`
  }, {
    path: path,
    cacheId: cacheId
  })

  app.component("RootComponent", RootComponent)
  app.component("TopNav", TopNav)
  app.component("WeatherPicker", WeatherPicker)
  // app.component("TimespanPicker", TimespanPicker)
  // app.component("TooltipsToggler", TooltipsToggler)
  app.component("MapGridComponent", MapGridComponent)
  app.component("MapRowComponent", MapRowComponent)
  app.component("MapComponent", MapComponent)
  app.component("HighchartComponent", HighchartComponent)

  app.mount(mountElt)
}