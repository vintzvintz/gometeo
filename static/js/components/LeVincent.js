
import { ref } from 'vue'

export const LeVincent = {

  props: {
    title: String,
  },

  setup() {
    const count = ref(0)
    return { count }
  },

  template: /*html*/`
<h1>{{title}} </h1>
<button @click="count++">
  You clicked me {{ count }} times.
</button>`
}


export const ElVincenzo = {

  props: {
    title: String,
  },

  setup() {
    const count = ref(0)
    return { count }
  },

  template: /*html*/`
<h1>{{title}} </h1>
<button @click="count++">
  ElVincenzo a été cliqué {{ count }} fois.
</button>`
}
