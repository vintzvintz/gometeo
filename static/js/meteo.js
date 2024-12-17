/****************************
 * Meteo kingsize
 * (c) 2020 Vincent Dandieu 
 **************************/
 
"use strict";

Vue.use(Vuex);

const theStore = new Vuex.Store({
    state: {
 
      // données fixes
      name: "default",         
      idtech: "",        // DEPT69, REGIN10, etc...
      taxonomy: "",      // DEPARTEMENT, REGION, PAYS
      svgmap: "",        // url du fond de carte SVG
      bbox: {},          // lat_S / lat_N / lng_O / lng_E
      
      // pour le breadcrumb
 //     parents: [],
  
      // Données interactives
      weathers: [
          { title: "Prévisions", class: "previsions", graph:'temperature', active: !0 },
          { title: "Vent", class: "vent", graph:'temperature', active: !1 },
          { title: "UV", class: "uv", graph:'temperature', active: !1 },
          { title: "Temp. ressentie", class: "ressenti", graph:'ressenti',active: !1 },
          { title: "Pression", class: "pression", graph:'pression',active: !1 },
          { title: "Couv. nuageuse", class: "couverture", graph:'couverture',active: !1 },
          { title: "Humidité", class: "humidite", graph:'humidite', active: !1 }
      ],

      // points d'intéret avec les prévisions associées
      prevs: [],
  
      // sous-zones de la carte ( sans objet pour les dept )
      subzones: [],   
      
      enabletooltips: true,

      timespans: [
        {title: '3 jours', active: true, rangs: [4,5,6,7, 8,9,10,11, 12,13,14,15, 16,17,18,19]},
        {title: 'Hier + aujourd\'hui', active: false, rangs: [0,1,2,3, 4,5,6,7] },
        {title: 'Semaine', active: false, rangs: [4,5,6,7, 8,9,10,11, 12,13,14,15, 16,17,18,19, 20,21,22,23, 24,25,26,27, 28,29,30,31] },
        {title: 'Long terme', active: false, rangs: [5,9,13,17,21,25,29,33,37,41,45,49,53,57] },
        {title: 'Tout', active: false, rangs: [0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37,38,39,40,41,42,43,44,45,46,47,48,49,50,51,52,53,54,55,56,57,58,59] },
      ]
    },
  
    getters: {
        get_name: function(state) { return state.name },
        get_idtech: function(state) { return state.idtech },
        get_taxonomy: function(state) { return state.taxonomy },
        get_bbox: function(state) { return state.bbox },
        get_prevs: function(state) { return state.prevs },
        get_chroniques: function(state) { return state.chroniques },
        get_subzones: function(state) { return state.subzones },
        get_weathers: function(state) { return state.weathers },
        get_weather: function(state) {
            for (let i in state.weathers)
               if (state.weathers[i].active) return i;
            return undefined
        },
        get_timespans: function(state) {return state.timespans},
        get_timespan: function(state) {
            return function() {
                for( let i in state.timespans ) 
                    if( state.timespans[i].active) return state.timespans[i].rangs;
                return !1
            }
        },
        get_enabletooltips: function(state) { return state.enabletooltips},
        get_weather_title: function(state) { return state.weathers.find( w => w.active ).title },
        get_weather_graph: function(state) { return state.weathers.find( w => w.active ).graph },
        get_image_path: function(state) { return  'svg/' + state.idtech.toLowerCase() + '.svg' },
    },
  
    mutations: {  
        SET_MAPDATA: function( state, data) {
              state.name = data.name;
              state.idtech = data.idtech;
              state.taxonomy = data.taxonomy;
              state.bbox = data.bbox;
              state.svgmap = data.svgmap;
              state.subzones = data.subzones;
              state.prevs = data.prevs;
              state.chroniques = data.chroniques;
        },
        SET_WEATHER: function(s, w) { 
            for (let i in s.weathers) { 
                s.weathers[i].active = w == i;
            }
        },
        SET_TIMESPAN: function(state, ts) {
            for (let i in state.timespans) {
                state.timespans[i].active = (ts==i)
            }
        },
        TOGGLE_TOOLTIPS: function( state, data ) {
            state.enabletooltips = !state.enabletooltips    
        },
    },

    actions: {
        set_mapdata: function( ctx, payload ) { ctx.commit( "SET_MAPDATA", payload )},
        set_weather: function( ctx, payload ) { 
          ctx.commit( "SET_WEATHER", payload )
        },
        set_timespan: function( ctx, payload) { ctx.commit( "SET_TIMESPAN", payload )},
        toggle_tooltips: function( ctx, payload ) { ctx.commit( "TOGGLE_TOOLTIPS", payload)},
    },
  })
  
var mapComponent =  {  

    store : theStore,

    template: '<div class="map_grid_item"><div class="titre_carte">{{get_weather_title}} - {{get_echeance}}</div>    \
             <div v-bind:id="get_mapid" class="map_component"></div></div>',
 
    props: {
        'rang': { required: true },
    },

    data: function() { 
        return {
            map: null,
            markers: [],
            updateControl: null
        }
    },
    beforeDestroy: function() {
        // the "unsubscribe" function is returned by the $store.subscribe()
        if( this.unsubscribeFromStore ) {
            this.unsubscribeFromStore();
        }
    },
    computed: {
        get_name: function() { return this.$store.getters.get_name },
        get_idtech: function() { return this.$store.getters.idtech },
        get_mapid: function() { return 'map-'+this.rang},
        get_image: function() { 
            var img = new Image;
            img.src = this.$store.getters.get_image_path;
            return img
        },
        get_bbox: function() { return this.$store.getters.get_bbox},
        get_subzones: function() { return this.$store.getters.get_subzones},
        get_prevs: function() {  return this.$store.getters.get_prevs[ this.rang ] },
        get_weather: function() { return this.$store.getters.get_weather },
        get_echeance: function() { return this.get_prevs && this.get_prevs.date_txt },
        get_weather_title : function() { return this.$store.getters.get_weather_title},
        get_enabletooltips : function() { return this.$store.getters.get_enabletooltips},
        get_show: function() { return true; }
    },

    mounted: function() {
        var t = this;

        // subscribe() returns a function to unsuscribe later (when component is destroyed)
        this.unsubscribeFromStore = this.$store.subscribe( function(mutation, state) {
            switch (mutation.type) {

                // we do not subscribe to SET_TIMESPAN in map-component because
                // 1- a unit map component does not need to know the timespan selected
                // 2- SET_MAPDATA is be triggered by timespan-picker 

                case "SET_MAPDATA" :
                    t.initMap();
               //   break;   
                case "SET_WEATHER":
                    t.updateMarkers();
                //  break;       
                case "TOGGLE_TOOLTIPS":
                    t.updateTooltipsVisibility();
            }
        });
        window.addEventListener( 'resize', function() {  
            t.map && t.map.invalidateSize() 
        })  
    },

    methods: {
        initMap: function() {
            //  when timespan changes, components are cached/re-used by v-for algorithm
            // so just skip initMap because map and subzones do not change.
            if (this.map) 
                return true;         

            let bounds = L.latLngBounds( [
                [ this.get_bbox.lat_S, this.get_bbox.lng_O ],
                [ this.get_bbox.lat_N, this.get_bbox.lng_E ]]);

             this.map = L.map(
                this.get_mapid, {
                    center: bounds.center,
                    fullscreenControl: !0,
                    cursor: !0,
                    scrollWheelZoom: !1,
                    zoomSnap: 1e-4,
                    zoomDelta: .1,
                    zoomControl: !1,
                    dragging: !1,
                    tap: !1,
                    maxBoundsViscosity: 1,
                    keyboard: !1,
                    doubleClickZoom: !1,
                    attributionControl: false,
                });

                this.updateControl = L.control.attribution( {prefix: ""} );
                this.updateControl.addTo(this.map);

            let overlay = L.imageOverlay(this.get_image, bounds);
            this.map.addLayer(overlay);
            this.map.setMaxBounds(bounds);
            this.map.fitBounds(bounds);
            this.map.setMinZoom(this.map.getBoundsZoom(bounds, !0))

            this.drawSubzones();
        },

        drawSubzones: function() {
            var t = this;

            var subZonesPane = this.map.createPane( 'subzones' );

            this.get_subzones.forEach( function(z) {
                let path = z.properties.prop_custom.path
                let nom = z.properties.prop_custom.name 
                L.geoJSON(
                    z, {
                        color: "transparent",
                        fillColor: "transparent",
                        weight: 3,
                        pane: subZonesPane,
                        onEachFeature: function(feature, layer) {
                           // layer.bindTooltip(nom, { direction: "auto" });
                            layer.on("mouseover", function() {
                                        this.setStyle({ color: "#FFF", fillColor: "transparent" }),
                                            layer.openPopup()
                                    }),
                                    layer.on("mouseout", function() {
                                        this.setStyle({ color: "transparent", fillColor: "transparent" }),
                                            layer.closePopup()
                                    }),
                                    layer.on("click", function() {
                                        window.location = path
                                    })
                            }
                        }
                    ).addTo(t.map);
            })
        },

        updateMarkers: function() {
            let pois = this.get_prevs && this.get_prevs.pois
            if (pois) {
                this.removeMarkers();
                pois.forEach( this.generateMarker );

                this.updateControl.setPrefix ("Màj : "+this.get_prevs.updated );
            }
        },

        updateTooltipsVisibility: function() {            
            var t = this;
            this.map && 
            this.map.getPane('markerPane') &&
            this.map.getPane('markerPane').childNodes.forEach( function(m) {
                let classes = m.classList
                t.get_enabletooltips ? 
                    classes.add('leaflet-interactive') :
                    classes.remove('leaflet-interactive')
                m.className= String(classes)
            })
        },

        removeMarkers: function() {
            let t = this;
            Object.keys(this.markers)
                    .forEach(function(mark) {
                        t.map.removeLayer(this[mark]);
                        delete this[mark]
                    },
                    this.markers)
        },

        round5: function(e) {
            return 5 * Math.ceil(e / 5)
        },

        generateMarker: function(prev, idx, all_prevs) {
            var t = this
            var weather = this.get_weather; // 0:prevision / 1:vent / 2:uv / etc...

            let marker_disabled = false;
            var ico = prev.weather_icon;
            var txt = prev.T;
            if( txt!==null) {
                // court terme
                (txt = parseFloat(txt), txt = Math.round(txt), txt += "°") 
            } else if (prev.T_min !== null && prev.T_max !==null) {
                // a long terme
                txt = ' <span class="tmin">' + Math.round(prev.T_min) + '°</span>' +
                        '/<span class="tmax">' + Math.round(prev.T_max) + '°</span>'
            } else {
                marker_disabled = true;
            }

            if ( weather == 1) {
              //  console.log( "Affichage du vent" );
                marker_disabled = (prev.wind_speed == null)
                if( !marker_disabled ) {
                    ico = prev.wind_icon;
                    if (-1 == ico ) {
                        ico = "Variable";
                    } 
                    let kmh = this.round5(3.6 * prev.wind_speed),
                        rafale = this.round5(3.6 * prev.wind_speed_gust);
                    txt = kmh;
                    if ( rafale >= 40 ) {
                        txt = '<span>' + kmh + '</span><span style="color:red;">|' + rafale + '</span>';
                    }
                }
            } else if( weather == 2 ) {
               // console.log( "Affichage des UV" );
                marker_disabled  = ( prev.uv_index == null );
                if( ! marker_disabled ){
                    ico = "UV_" + prev.uv_index, txt = "";
                }
            } else if( weather == 3 ) {
              //  console.log( "Temperature ressentie" );
                marker_disabled  = ( prev.T_windchill == null) 
                if( !marker_disabled ) {
                    let temp_r = Math.round(parseFloat(prev.T_windchill));
                    txt  = '<span style="color: brown;">' + temp_r + '</span>';
                }

            } else if( weather == 4 ) {
               // console.log( "Pression" );   
                marker_disabled = ( prev.T_windchill == null);
                if ( !marker_disabled) { 
                    (txt = parseFloat(prev.P_sea), txt = Math.round(txt), txt += "")     
                }

            } else if( weather == 5 ) {
                //console.log( "Couverture nuageuse" );            
                marker_disabled = ( prev.total_cloud_cover == null);
                if( !marker_disabled) { 
                    txt = prev.total_cloud_cover + "%";
                }

            } else if( weather == 6 ) {
                let hr_unit = "%";
                if( prev.relative_humidity !== null) {
                    // court terme
                    (txt = parseFloat(prev.relative_humidity), txt = Math.round(txt), txt += hr_unit) 
                } else if  (prev.relative_humidity_min !== null) {
                    // a long terme
                    txt =   '<span class="hr_min">' + Math.round(prev.relative_humidity_min) + hr_unit + '</span>' +
                            '/<span class="hr_max">' + Math.round(prev.relative_humidity_max) + hr_unit + '</span> ';
                } else {
                    marker_disabled = true;
                }

            } else if( weather != 0) { 
                console.log( "erreur : mode d'affichage inconnu" );
                marker_disabled = true;
            }
             
            let pos_v = (prev.lat > (this.get_bbox.lat_N + this.get_bbox.lat_S)/2 ) ? 'Top' : 'Bottom';
            let pos_h = (prev.lng > (this.get_bbox.lng_O + this.get_bbox.lng_E)/2 ) ? 'Right' : 'Left';
            var pos = pos_v+' '+pos_h;

            !marker_disabled && this.add_marker({
                path: "",
                T: Math.round(prev.T),
                icon: ico,
                icon_text: txt,
                lat: prev.lat,
                lng: prev.lng,
                title: prev.title,
                desc: prev.weather_description,
                min: Math.round(prev.T_min),
                max: Math.round(prev.T_max),
                position: pos,
/*                weather_confidence_index: prev.weather_confidence_index,*/
                currentKey: idx,
                totalLength: all_prevs.length
            });
        },

        add_marker: function(e) {
            var t = this;

            if (e.title.toLowerCase() in this.markers) {
                console.log("Warning  : ajout d'un marker déja présent")
            }

            let weather = this.get_weather;
            //this.default_size = 50,
            this.default_size = 40;
            if ( this.get_weather == 1)  // icones du vent plus petites
                this.default_size = 25;

            /*  
            // ajuste la taille des icones pour les mobiles
            if( 1 == s && "mobile" == this.get_device ?
                this.default_size = 20 :
                1 != s && "mobile" != this.get_device ||
                (this.default_size = 30);
            */

            // style pour le texte température min/max (qui contient un slash)
            let icon_text_style = "";
            if( ("string" == typeof e.icon_text || e.icon_text instanceof String) && 
                    e.icon_text.indexOf("/") > -1 ) {
                (icon_text_style = "font-size: 12px;");
            }
//            let elt_a = '<a href="' + e.path + '"><img src="svg/' + e.icon + '.svg" alt="' + e.desc +

//            let elt_a = '<a href="' + e.path + '">' +
            let elt_a = '<a>'+
                '<img src="svg/' + e.icon + '.svg" alt="' + e.desc +
    
                '" title="' + e.desc +
                '" class="icon shape-weather" style="width: ' + this.default_size + 'px"/>';
            if( e.icon_text && "" != e.icon_text && "NaN°" != e.icon_text ) {
                (elt_a += '<span class="icon_text" style="' + icon_text_style + '">' + e.icon_text + "</span>")
            }
            elt_a += "</a>";

            let div_icon = !1;
            div_icon = L.divIcon({
                html: elt_a,
                className: "iconMap-1",
                iconSize: [t.default_size, t.default_size],
                iconAnchor: [t.default_size / 2, t.default_size / 2]
                });

            let mark = L.marker([e.lat, e.lng], { icon: div_icon }).addTo(this.map);


            //  Ajoute le tooltip 
            let tt_direction = !1;
            let tt_offset = L.point(0, 10);

            switch (e.position) {
                case "Top Left":
                    tt_offset = L.point(100, 10), tt_direction = "bottom";
                    break;
                case "Top Right":
                    tt_offset = L.point(-100, 10), tt_direction = "bottom";
                    break;
                case "Bottom Left":
                    tt_offset = L.point(100, 10), tt_direction = "top";
                    break;
                case "Bottom Right":
                    tt_offset = L.point(-100, 10), tt_direction = "top";
                    break;
                default:
                    tt_offset = L.point(0, 10), tt_direction = "auto";
            }

            let T_min_max = "<p>Min : <span class='temp-min'>" + e.min +
            "°</span> Max : <span class='temp-max'>" + e.max +
            "°</span></p>";
            if (void 0 === e.min) {
                T_min_max = "";
            }
            /*
            let indice = "";
            if( null !== e.weather_confidence_index &&  void 0 !== e.weather_confidence_index ) {
                (indice = "<p>Indice de confiance: " + e.weather_confidence_index + "/5</p>");
            }
            */
            let tt_html = "<div class='map_tooltip'><h3 class='map_tooltip_location'>" + e.title +
                '</h3><img src="svg/' + e.icon +
                '.svg" class="icon shape-weather" alt="' + e.desc +
                '" title="' + e.desc +
                '" style="width: ' + t.default_size + 'px"/>';

            if( e.icon_text && "" != e.icon_text && "NaN°" != e.icon_text ) {
                tt_html += "<div class='map_tooltip_temp'>" + e.icon_text + "</div>"
            }

            tt_html += "<p class='map_tooltip_description'>" + e.desc + "</p>" + T_min_max /*+indice*/ + "</div>";
            
            mark.bindTooltip( tt_html, {
                sticky: false,
                direction: tt_direction,
                offset: tt_offset,
            })
            this.markers[e.title.toLowerCase()] = mark;
        }      
    }  
};


var weatherPickerComponent = {
    store : theStore,
    data: function() { return { activeItem: 0 } },
    render : function( createElt ) {
        var t = this;
        return createElt( "div", { class:"data_picker" }, [
                  createElt( "ul", t.get_weathers.map( function (item, index, a) {
                        return createElt( "li", 
                        {   class: [ item.class ],
                            attrs: { },
                            on: { click : function( ) {
                                t.setActive(index);
                                t.set_weather(index);
                            }},
                            key: "key-"+index,
                        }, [createElt("a", 
                            { attrs: {"href":"#"},
                              class: [ { "active": t.isActive(index) }]
                            }, 
                            item.title )] )
                    }
                ))])
    },
    mounted: function() {
        var e = this;
        this.$store.subscribe(function(t, s) {
            switch (t.type) {
                case "SET_WEATHER":
                    e.setActive(t.payload)
            }
        })
    },
    computed: {
        get_weathers: function() { return this.$store.getters.get_weathers },
    },
    methods: {
        set_weather: function(e) { this.$store.dispatch("set_weather", e )},
        isActive: function(e) { return this.activeItem === e },
        setActive: function(e) { this.activeItem = e },
    }
};

var tooltipsToggler = {
    store: theStore,
    template: '<div id="tooltip_toggler" v-on:click="toggle"><a :class="css_class" href="#">Tooltips : {{status}}</a></div>',
    computed: {
        status: function() { return this.$store.getters.get_enabletooltips ? "Oui" :"Non" ; },
        css_class: function() { return this.$store.getters.get_enabletooltips ? "active" : "" },
    },
   methods: {
       toggle: function( ) { this.$store.dispatch ( "toggle_tooltips") },
   },
};


var timespanPickerComponent ={
    store : theStore,
    data: function() { return { activeTimespan: 0 } },
    render : function( createElt ) {
        var t = this;
        return createElt( "div", { class:"timespan_picker" }, [
            /*      createElt( "div", {attrs: {id:"idTSP"} }, "Titre du timespan picker"), */
                  createElt( "ul", t.get_timespans.map( function (item, index, a) {
                        return createElt( "li", 
                        {   on: { click : function( ) {
                                t.setActive(index);
                                t.set_timespan(index);
                            }},
                            key: 'tp-'+index,
                        }, [createElt("a", 
                            { attrs: {"href":"#"}, 
                              class: [ { "active": t.isActive(index)  }]
                            }, 
                            item.title )] )
                    }
                ))])
    },
    mounted: function() {
        var t = this;
        this.$store.subscribe(function(mutation, s) {
            switch (mutation.type) {
                case "SET_TIMESPAN":
                    t.setActive(mutation.payload)
                    // trigger init ( or re-init ) on map components after v-for has finished updating the DOM
                    Vue.nextTick(function() {t.$store.dispatch('set_mapdata', window.globalMapData) }) ;
                    break;
            }
        })
    },
    computed: {
        get_timespans: function() { return this.$store.getters.get_timespans },
    },
    methods: {
        set_timespan: function(e) { this.$store.dispatch("set_timespan", e ) },
        isActive: function(e) { return this.activeTimespan == e },
        setActive: function(e) { this.activeTimespan = e },
    }
};

var highChartComponent = {
    store : theStore,
    template: '<div v-bind:id="container_id"></div>',

    data: function() { return {
        chart: null,
        container_id: 'graph_container',
    }},

    computed: {
        get_chroniques: function() { return this.$store.getters.get_chroniques },
        get_weather: function () {return this.$store.getters.get_weather },
        get_conf: function() { 
            let confs = { 
                temperature: {
                    title: 'Temperature',
                    axeY1: '°C',
                    series: { 
                        'T' : { lineWidth:1, color:'#BBB' },
                        'T_max' : { lineWidth:1, color:'#D11' },
                        'T_min' : { lineWidth:1, color:'#22D' },
                    },
                },
                pression:  {
                    title: 'Pression au niveau de la mer',
                    axeY1: 'hPa',
                    series: { 
                        'P_sea' : { lineWidth:1, color:'#222' },
                    },
                    
                },
                ressenti: {
                    title: 'Température ressentie',
                    axeY1: 'indice de refroidissement',
                    series: { 
                        'T_windchill' : { lineWidth:1, color:'#444' },
                    },

                },
                couverture: {
                    title: 'Couverture nuageuse',
                    axeY1: '%',
                    series: { 
                        'total_cloud_cover' : { lineWidth:1, color:'#222' },
                    },
                }, 
                humidite: {
                    title: 'Humidité relative',
                    axeY1: '%',
                    series: { 
                        'relative_humidity' : { lineWidth:1, color:'#BBB' },
                        'relative_humidity_max' : { lineWidth:1, color:'#1D1' },
                        'relative_humidity_min' : { lineWidth:1, color:'#DD1' },
                    },
                },
            }
            return confs[ this.$store.getters.get_weather_graph ];
        }
    },

    mounted: function() {
        var t = this;
        this.chart || (t.initGraph(), t.updateGraph()) ;
        this.$store.subscribe( function(mutation, x) {
            switch (mutation.type) {
                case "SET_MAPDATA" :
                    t.initGraph();
            //   break;   
                case "SET_WEATHER":
                    t.updateGraph();
            }
        })
    },

    methods: { 
        initGraph: function() {
            this.chart = Highcharts.chart(this.container_id, {
                chart: { 
                    type: 'spline',
                },
                plotOptions: {
                    series: {
                      color: '#666699',
                      enableMouseTracking: false,
                      marker: { enabled: false },
                      showInLegend: false,
                      states: { hover: {enabled: false} }
                    },
                    spline: { lineWidth: 1,turboThresold: 3 }
                },
                tootip: { enabled: false  },        
                xAxis: { 
                    id: 'axeX',
                    type: 'datetime',
                    dateTimeLabelFormats: { day:'%e %b' },
                    floor: Date.now() - 24*3600*1000,
                    plotLines: [{ 
                      color: '#FC0FC5',
                      value: Date.now(),
                      width: 2,
                      zIndex: 12,
                    }],    
                },
            })
        },

        updateGraph: function() {
            var t = this;

            // remove all previous series and Y-axis but do not redraw now
            while(this.chart.series.length ) { this.chart.series[0].remove(false) }
            this.chart.axes.forEach( (axe) => { axe.isXAxis || axe.remove(false) } );

            var conf = this.get_conf;
            this.chart.setTitle( {text: conf.title}, {}, false );   // no subtitle, no redraw
            this.chart.addAxis(  {id:'axeY1', title : {text: conf.axeY1}}, false, false ) // no Xaxis, no redraw

            for (let s in conf.series) {
                Object.values( this.get_chroniques[s] ).forEach( function(serie_data,i) {
                    let opts = { 
                        data: serie_data,
                    }
                    Object.assign( opts, conf.series[s]);
                    t.chart.addSeries( opts, false );
                })
            }
            this.chart.redraw();
        },
    },
};


// instance principale Vue. 
new Vue({
  el: '#vuejs_root',
  store: theStore,
  components: {
    'data-picker':      weatherPickerComponent,
    'map-component':    mapComponent,
    'tooltips-toggler': tooltipsToggler,
    'timespan-picker':  timespanPickerComponent,
    'highchart-graph':  highChartComponent,
  },
  data: {
      loading: !0,
  },

  computed: {
    get_rangs: function() { return this.$store.getters.get_timespan() },
    display_graph: function() { 
        return this.$store.getters.get_taxonomy == "DEPARTEMENT" || this.$store.getters.get_taxonomy == "REGION"
    },
  },
  mounted: function() {
      this.main();
  },
  methods: {
        main: function() {
          var t=this;
          this.$store.dispatch("set_mapdata", window.globalMapData );  
          //Vue.nextTick(function() {t.$store.dispatch('set_mapdata', window.globalMapData) }) ;
        },
  },
});    //  new Vue({...})


