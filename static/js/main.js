
import { createApp, ref } from 'vue'

import {LeVincent, ElVincenzo} from './components/LeVincent.js'

const app = createApp({
  setup() {
    const message = ref("Le createApps message")
    return { message }
  },

  template: /*html*/`
  <LeVincent title="message"></LeVincent>
  <LeVincent2 title="weshito"></LeVincent2>`
})


app.component("LeVincent", LeVincent)
app.component("LeVincent2", ElVincenzo)


app.mount('#vuejs_root')
