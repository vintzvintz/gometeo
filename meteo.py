#!/usr/bin/python3

import codecs
import requests
import string
import re
import json
import certifi
import os
import os.path
import pickle
import logging
import locale
import inspect
import sys
import argparse
from datetime import date, datetime, time, timedelta, timezone
import time as time_utils
import pytz
from  multiprocessing import Process
from html.parser import HTMLParser
from threading import Lock, RLock
import concurrent.futures

from svgcrop import SvgCropper

DIR_WWW = 'www/'
DIR_SVG = os.path.join(DIR_WWW, 'svg/')
URL_SVG = 'svg/'
DIR_JSON=os.path.join(DIR_WWW, 'data/')
URL_JSON='data/'
DIR_CACHE = 'cache/'
HTML_TEMPLATE = 'templates/template.html'

HTTP_METEOFRANCE = 'https://meteofrance.com'
USER_AGENT = 'Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:79.0) Gecko/20100101 Firefox/79.0'

FORECAST_PROPERTIES = [ 
  'moment_day','time','T','T_windchill','relative_humidity', 'P_sea',    
  'wind_speed','wind_speed_gust','wind_direction','wind_icon',
  'weather_icon','weather_description', 'total_cloud_cover'
]
FORECAST_PROPERTIES_DAILY = [
  'T_min', 'T_max', 'uv_index', 
  'relative_humidity_min', 'relative_humidity_max',
  'daily_weather_icon', 'daily_weather_description'
]
FORECAST_CHRONIQUES = [ 
  'T', 'T_min', 'T_max', 'T_windchill', 'P_sea', 'total_cloud_cover',
  'relative_humidity',  'relative_humidity_min', 'relative_humidity_max'
]
FORECAST_CHRONIQUES_MAXDAYS = 10

# tronque les fonds de carte ( en % de la dimension initiale )
CROP_MAP_N = 0.08
CROP_MAP_S = 0.08
CROP_MAP_E = 0.08
CROP_MAP_O = 0.20

ZTIME_REGEX = re.compile( r'^(?P<year>\d\d\d\d)-(?P<month>\d\d)-(?P<day>\d\d)T(?P<hour>\d\d):(?P<min>\d\d):(?P<sec>\d\d)\.000Z$')
TZ_PARIS = pytz.timezone("Europe/Paris")

#  Helper for lame authentication
class MyAuth(requests.auth.AuthBase):
  def __init__(self, token):
    self.token=token

  def __call__(self, r):
    r.headers["Authorization"] = "Bearer "+self.token
    return r

class MapParser(HTMLParser):
  #  find inline json content and find relevant data
  # <script type="application/json" data-drupal-selector="drupal-settings-json"> 
  def __init__(self):
    super().__init__()
    self.isInDrupalSettings = False

  def handle_starttag(self, tag, attrs):
    if(tag.lower() == "script"):
      # convert list of pairs to a dict 
      attr_dict = {a[0]:a[1] for a in attrs}
      if (     "type" in attr_dict 
            and attr_dict["type"] == "application/json" 
            and "data-drupal-selector" in attr_dict 
            and attr_dict["data-drupal-selector"] == "drupal-settings-json" ):
        self.isInDrupalSettings = True 

  def handle_endtag(self, tag):
    if(tag == "script"):
      self.isInDrupalSettings = False

  def handle_data(self, data):
    if (self.isInDrupalSettings):
      self.json_data = json.loads(data)

class Mf_map:
  # Counterpart of MF forecast pages, with geographical, svg and forecast data
  def __init__(self, data, own_path, parent):
    conf = data["mf_tools_common"]["config"] 
    self.api_url="https://"+conf["site"]+"."+conf["base_url"]    # https://rpcache-aa.meteofrance.com/internet2018client/2.0
    self.infos = data["mf_map_layers_v2"]
    self.pois = data["mf_map_layers_v2_children_poi"]
    self.subzones = data["mf_map_layers_v2_sub_zone"]
    self.own_path = own_path
    self.parent = parent
  
  def breadcrumb(self):
    c = {'path': self.own_path+'.html', 'name':self.infos['name']}
    if( self.parent ):
      return self.parent.breadcrumb() + [ c ] 
    return [ c ]

  def get_related_data(self, session, cache_data, cache_assets):
    self.prevs       = self.get_forecast( session, cache_data )
    self.pictos      = self.get_pictos( session, cache_assets )
    self.geography   = self.get_geography( session, cache_assets )
    self.svgmap      = self.get_svgmap( session, cache_assets )

  def build_json(self):
    obj = {}
    obj["name"] = self.infos['name']
    obj["idtech"] = self.infos['field_id_technique']
    obj["taxonomy"] = self.infos['taxonomy']  

    obj["subzones"] = self.geography["features"]
    lng_O = self.geography["bbox"][0]
    lat_S = self.geography["bbox"][1]
    lng_E = self.geography["bbox"][2]
    lat_N = self.geography["bbox"][3]

    # coordonnées recalculées pour la carte rognée
    obj["bbox"] = { 
       'lng_O': lng_O + CROP_MAP_O*(lng_E-lng_O),
       'lat_S': lat_S + CROP_MAP_S*(lat_N-lat_S),
       'lng_E': lng_E - CROP_MAP_E*(lng_E-lng_O),
       'lat_N': lat_N - CROP_MAP_N*(lat_N-lat_S)
     }

    prevs = self.crunch_prevs()
    # ajoute les prévisions du passé
    with PrevCache( obj["idtech"] ) as cache:
      cache.update( prevs )

    # convertit en array car les datetime python ne sont pas supportés en JSON
    obj['prevs']=[]
    for ech in sorted( prevs.keys() ):
      obj['prevs'].append( prevs[ech] )

    # ajoute les données pour les graphiques
    # c'est redondant mais évite de faire la conversion de structure côté client...
    obj['chroniques'] = self.build_graphdata( prevs )

    write_to_dir( "\"use strict\";\n var globalMapData = "+json.dumps(obj), 
                  obj["idtech"] + "-data.js", 
                  DIR_JSON )

  def crunch_echeances(self):
    # liste sans doublon des horodates disponibles pour tous les lieux
    # et correspondance entre les echéances ( 30/8/2020 18h) et l'instant  (matin/am/soir/nuit)
    # les résultats sont mis en cache car la méthode est appelée plusieurs fois
    if ( not hasattr(self, '_echeances_uniques') or not hasattr(self, '_moments_day') ) :     
      echeances_all = []
      moments_day = {}

      for lieu in self.prevs:
        echeances_lieu = set()
        for forecast in lieu["properties"]["forecast"]:    
          ech = forecast['time']
          echeances_lieu.add( ech )
          moments_day[ech] = forecast['moment_day']
        echeances_all.append( echeances_lieu )

      # intersection des echéances de chaque lieu
      from functools import reduce
      self._echeances_uniques = sorted(reduce( lambda x,y: x&y, echeances_all ))
      self._moments_day = moments_day

    return self._echeances_uniques , self._moments_day
    
  def crunch_prevs(self):    
    # inversion de la structure :
    # l'API  MF revoie une liste de lieux contenant toutes les échéances
    # on construit une liste d'échéances contenant des lieux...
    all_prevs = {}
    all_echeances, moments_day = self.crunch_echeances()
    for ech in all_echeances :    

      # données pour 1 carte élémentaire
      # prévisions pour une échéance particulière
      data = {     
        'date_iso': ech,
        'date_txt': parse_ztime(ech).astimezone(TZ_PARIS).strftime("%A %d %b %Hh"),
        'updated' : None,  # update_time est défini au niveau du lieu et non de la carte.
        'moment'  : moments_day[ech],
        'pois' : [],
      }

      for poi in self.pois:                         # POI du "drupalSettings"
        for lieu in self.prevs:         # POI renvoyés par l'API previsions
          props = lieu['properties']
          if( poi["insee"] == props["insee"] ):

            # update_time est défini au niveau du lieu et non de la carte.
            # mise a jour seulement au premier passage
            if( not data['updated'] ):
              upd = parse_ztime( lieu['update_time']).astimezone( TZ_PARIS )
              data['updated'] = upd.strftime( '%a %d/%m %Hh%M')

            # init à partir du "drupal settings"
            # complété avec les prévisions
            poi_with_prev = { "title": poi["title"], \
                              "lat" : float(poi["lat"]), \
                              "lng" : float(poi["lng"]), \
                            }

            # insère les prévisions de l'instant
            for forecast in props["forecast"]:
              if( forecast["time"] == ech ):
                for p in forecast :
                  if p in FORECAST_PROPERTIES:
                    poi_with_prev[p] = forecast[p]

            # insère les prévisions journalières. Utiles pour remplacer T ou weather_icon après J+7
            for daily in props["daily_forecast"]:
              #la date correspond aux 10 premiers caractères
              if( daily["time"][0:10]  == ech[0:10] ):
                for p in daily :
                  if p in FORECAST_PROPERTIES_DAILY:
                    poi_with_prev[p] = daily[p]

            data['pois'].append( poi_with_prev )
            break    # if( poi["insee"] == lieu["insee"] )

      all_prevs[ech] = data
    return all_prevs

  def get_forecast( self, session, from_cache ):
    coords = [ "%s,%s"%(poi['lat'],poi['lng']) for poi in self.pois]
    params = { 'bbox'       : '',
               'coords'     : '_'.join(coords),
               'instants'   : 'morning,afternoon,evening,night',
               'begin_time' : '', 'end_time'   : '', 
               'time'       : ''
              }
    txt = session.get( self.api_url+'/multiforecast', from_cache=from_cache, params=params )
    return json.loads( txt )['features']

  def get_geography(self, session, from_cache):
    url = HTTP_METEOFRANCE + "/modules/custom/mf_map_layers_v2/maps/desktop/" \
          + self.infos['path_assets'] + "/geo_json/" \
          + self.infos["field_id_technique"].lower() + "-aggrege.json"
    geo_data = json.loads( session.get( url, from_cache=from_cache ) )

    # supprime les sous-zones non référencées dans le "drupalSettings"
    # et ajoute les infos pour les sous-zones cliquables ( prop_custom )
    selected_zones = []
    if( len(self.subzones) ):
      for zone in geo_data['features'] :
        cible = zone['properties']['prop0']['cible']
        if( cible in self.subzones.keys() ):
          # '/previsions-meteo-france/hauts-de-france/1'
          fullpath = self.subzones[cible]["path"]
          m = re.search( r'/([\w-]*)/\w*$', fullpath )
          prop_custom = { "path": "%s.html" % m[1] ,
                          "name": self.subzones[cible]["name"] }
          zone['properties']['prop_custom'] = prop_custom
          selected_zones.append( zone )
    geo_data['features'] = selected_zones
    return geo_data

  def get_svgmap(self, session, from_cache):
    filename = self.infos["field_id_technique"].lower() + ".svg"
    url = HTTP_METEOFRANCE + "/modules/custom/mf_map_layers_v2/maps/desktop/" \
          + self.infos['path_assets'] + "/" + filename

    svg_original = session.get( url, from_cache=from_cache )
    svg = SvgCropper( svg_original )
    svg.crop( crop_O=CROP_MAP_O, crop_E=CROP_MAP_E,crop_N=CROP_MAP_N, crop_S=CROP_MAP_S)
    write_to_dir( str(svg), filename, DIR_SVG )
    return filename
        
  def get_pictos( self, session, from_cache ):
    icons = set()   # built-in duplicate removal :)
    for lieu in self.prevs :
      for daily in lieu['properties']['daily_forecast'] + lieu['properties']['forecast'] :
        for icon_name in ['daily_weather_icon', 'weather_icon', 'wind_icon', 'uv_index'] :
          try: 
            if( icon_name == 'uv_index') : 
              icons.add( 'UV_'+str(daily[icon_name]) )
            else:  
              icons.add( daily[icon_name] )
          except KeyError:
            pass

    icons_unique = [ i+'.svg' for i in icons if i!=None ]

    for name in icons_unique :
        pic_path =  "/modules/custom/mf_tools_common_theme_public/svg/weather/" + name 
        write_to_dir( session.get( HTTP_METEOFRANCE+pic_path, from_cache=from_cache ), name, DIR_SVG ) 
    return icons_unique      

  def build_graphdata(self, prevs):
    g = {}
    for ech, prev in prevs.items() :
      ech_obj = parse_ztime(ech)
      if ( ech_obj - datetime.now(timezone.utc) < timedelta( days = FORECAST_CHRONIQUES_MAXDAYS) ):

      # https://docs.python.org/3/library/datetime.html#datetime.datetime.timestamp
      # timestamp in miliseconds for Javascript Date()
        ts = int((ech_obj - datetime(1970, 1, 1, tzinfo=timezone.utc))/timedelta(milliseconds=1))

        for poi in prev['pois'] :
          lieu = poi['title']
          data = { k:v for k,v in poi.items() if (k in FORECAST_CHRONIQUES and v) }
          for serie, val in data.items():    # serie = 'T', 'T_min', 'T_max', etc...
            if ( not lieu in g.keys() ) :
              g[lieu]= { k:{} for k in FORECAST_CHRONIQUES }
            g[lieu][serie][ts] = val

    # transforme les dict en arrays utilisables directement pour highcharts      
    all_series={}
    for lieu, series in g.items() :
      for nom, serie_dict in series.items() :
        serie_array = [ [ts,serie_dict[ts] ] for ts in sorted(serie_dict.keys()) ]
        try:
          all_series[nom].append( serie_array )
        except KeyError:
          all_series[nom] = [ serie_array ]

    return all_series

  def build_html( self, templatefile = HTML_TEMPLATE ) :
    with open(templatefile, 'r') as f:
      tmpl = string.Template( f.read() )

    breadcrumb = [ '<a href="%s">%s</a>' % ( e['path'], e['name'] ) for e in self.breadcrumb()]

    data = {
      'head_description': "Prévisions météo "+self.infos['name']+" en grand format.",
      'head_title': "Météo %s monopage" % self.infos['name'],
      'breadcrumb' : "\n".join(breadcrumb),
      'idtech' : self.infos['field_id_technique'],
    }
    write_to_dir( tmpl.substitute( data ), self.own_path+'.html', DIR_WWW ) 

class PrevCache:
  def __init__(self, idtech ):
    self.idtech = idtech
    self.cache_path = os.path.join( DIR_CACHE, idtech+"-cache.dat" )
    self._cache= { }

  def __enter__(self):
    logging.debug("Enter PrevCache (%s)" % self.cache_path)
    if( os.path.isfile( self.cache_path ) ):
      with open( self.cache_path, 'rb' ) as f :
        self._cache = pickle.load( f )

    aujourdhui = date.today()
    hier = aujourdhui - timedelta( days=1)

    self.add_missing_hiver( hier , "matin", 9)
    self.add_missing_hiver( hier, "après-midi", 15)
    self.add_missing_hiver( hier, "soirée", 21)
    self.add_missing_hiver( aujourdhui, "nuit", 3)
    self.add_missing_hiver( aujourdhui, "matin", 9)
    self.add_missing_hiver( aujourdhui, "après-midi", 15)
    self.add_missing_hiver( aujourdhui, "soirée", 21)

    # self.add_missing_utc( hier , "matin", 6)
    # self.add_missing_utc( hier, "après-midi", 12)
    # self.add_missing_utc( hier, "soirée", 18)
    # self.add_missing_utc( aujourdhui, "nuit", 0)
    # self.add_missing_utc( aujourdhui, "matin", 6)
    # self.add_missing_utc( aujourdhui, "après-midi", 12)
    # self.add_missing_utc( aujourdhui, "soirée", 18)


    return self

  def __exit__(self, exc_type, exc_val, exc_tb):
    with open( self.cache_path, 'wb') as f:
      pickle.dump(self._cache, f)
    return False  # do not suppress exceptions

  def add_missing_hiver( self, jour, moment, heure):

    # EDIT 2024/08/17 : ceci semble avoir changé, maintenant les prévisions sont à 0/6/12/18 heures UTC ??
    
    # les horodates "Z" de la source ne sont pas vraiment en UTC mais en heure locale constante.
    # càd 7/13/19/1 en été et 8/14/20/2 en hiver, pour avoir toujours 3/9/15/21 affichés.
    missing_datetime =  datetime.combine( jour, time(hour=heure) )
    missing_datetime_local = TZ_PARIS.localize( missing_datetime )
    missing_datetime_utc = missing_datetime_local.astimezone( pytz.utc )
    missing_ech = missing_datetime_utc.astimezone( pytz.utc ).strftime( '%Y-%m-%dT%H:%M:%S.000Z' )

    if( not missing_ech in self._cache.keys() ):
      #ajoute un élement vide pourl'echéance manquante
      self._cache[ missing_ech ] = { 
        'date_iso': missing_ech,
        'date_txt': missing_datetime.astimezone(TZ_PARIS).strftime("%A %d %b %Hh"),
        'updated' : None, 
        'moment'  : moment,
        'pois' : [],
      }


  def add_missing_utc(self, jour, moment, heure):
    """ ajoute un élement vide pour l'echéance manquante
    """
    missing_datetime =  datetime.combine( jour, time(hour=heure), pytz.utc )
    missing_ech = missing_datetime.strftime( '%Y-%m-%dT%H:%M:%S.000Z' )

    if( not missing_ech in self._cache.keys() ):
      self._cache[ missing_ech ] = { 
        'date_iso': missing_ech,
        'date_txt': missing_datetime.astimezone(TZ_PARIS).strftime("%A %d %b %Hh"),
        'updated' : None, 
        'moment'  : moment,
        'pois' : [],
      }


  def update( self, prevs ) :
    ce_matin = datetime.combine( datetime.now().date() , time(hour=5), timezone.utc )
    hier_matin = ce_matin - timedelta( days=1 )
    demain_matin = ce_matin + timedelta( days=1 )
    
    # enlève les vieilles previsions
    new_cache = { ech:prev for (ech,prev) in self._cache.items() if ech >= hier_matin.isoformat()}

    # ajoute ou met à jour les echéances courantes dans le cache
    for p in prevs :
      if (p < demain_matin.isoformat()):
        new_cache[p] = prevs[p]
    self._cache = new_cache

    # fusionne les données du cache dans le dict des prevs courante
    prevs.update(self._cache)

 # Manage session, connection pooling, cookies, auth token, etc...
class Mf_Crawler:

  def __init__(self, cache_data=False, cache_assets=True, test_mode=False):
    self.session = MF_Session()
    self.session.verify =  certifi.where() 
    self.session.headers.update( {"User-agent": USER_AGENT} )
    self.map_count = 0
    self.cache_data   = cache_data
    self.cache_assets = cache_assets
    self.test_mode = test_mode

    self.maps_fut = {}
    self.lock = Lock()
    
  def submit_map( self, ex, path, parent):
    with self.lock :
      f = ex.submit( self.get_map, executor=ex, path=path, parent=parent )
      self.maps_fut[f]=path
      debug("Submit %s. Total actif=%d" % (path, len(self.maps_fut)) )

  def run_threads( self, workers=None):
    # open a root context to avoid multiple enter/exit when only one worker 
    with self.session :     
      with concurrent.futures.ThreadPoolExecutor(max_workers=workers) as ex :
        # initialise avec la carte de France
        self.submit_map( ex=ex, path='/', parent=None )
        while len(self.maps_fut):
          with self.lock:
            futs = self.maps_fut.copy()
          done, not_done = concurrent.futures.wait( futs )
          # self.maps_fut est modifié pendant le wait(), des sous-zones sont ajoutées
          with self.lock :
            for f in done:
              self.maps_fut.pop(f)
            debug( "Tasks : %d done, %d pending" %(len(done), len(self.maps_fut)) )  
      debug("Executor ended")

  def get_map(self, executor, path, parent=None ):
    url = HTTP_METEOFRANCE + path
    self.map_count += 1

    with self.session as s:
      info( "Loading map #%d %s"% (self.map_count,url) )
      resp = s.get( url )

      # rpcache-aa API requires an auth token, which is just the obfuscated mfsession cookie.
      s.set_auth ( codecs.encode( s.cookies["mfsession"], "rot_13") )
      
      # extract json data embedded in HTML, do API calls and download svg static assets
      mp  = MapParser()
      mp.feed( resp )
      mp.close()

      mf_map = Mf_map( data=mp.json_data, parent=parent, own_path=self.own_path(url, path) )

      # launch recursion before getting related data
      sous_zones = self.children_paths( mf_map.subzones )
      for child_path in sous_zones :
        if executor :
          # ajouter les subzones dans la queue de l'executor 
          self.submit_map( ex=executor, path = child_path, parent = mf_map )
        else :
          # recursion classique single-thread
          self.get_map(executor = executor, path = child_path, parent = mf_map )

      mf_map.get_related_data( s, cache_data=self.cache_data, cache_assets=self.cache_assets )

    # restructure data as one big json file for the frontend
    mf_map.build_json()
    mf_map.build_html( HTML_TEMPLATE )

  def own_path(self, url, path):
  # l'url n'est pas dispo dans le html, il faut la faire suivre à part
    m = re.search( r'/([\w-]*)/\w*$', url )
    if (m):
      return m[1]
    elif ( path=="/" ) :
      return 'france'
    else:
      raise Exception( 'Erreur de nom ou de chemin')

  def children_paths( self, subzones ):
    try: 
      return [info['path'] for idtech,info in subzones.items() if self._map_filter(idtech) ]
    except AttributeError:
      return []

  # Une seule région en mode test
  def _map_filter( self, id ) :
    lid = id.lower()
    if( lid in ['dept988' ]         # nvelle calédonie
        or  lid.startswith('opp')   # zones spéciales ( montagne) 
        or (self.test_mode and lid.startswith('regin') and lid != "regin10") ):
      return False
    return True


class MF_Session (requests.Session ) :
  def __init__(self, cache_file='mf_cache.dat', cache_enable=True):
    super().__init__()
    self.assets = {}
    self.cache_path = os.path.join( DIR_CACHE, cache_file )
    self.cache_enabled = cache_enable
    self.refcount    = 0
    self._hitcount   = 0
    self._misscount  = 0
    self.ctx_lock    = Lock()
    self.auth_lock   = Lock()
    
  def __enter__(self):
    with self.ctx_lock :
      if( self.cache_enabled and os.path.isfile(self.cache_path) and self.refcount==0):
        with open( self.cache_path, 'rb' ) as f :
          self.assets = pickle.load( f )
          info( "MF_session : loaded %s assets from %s" % (len(self.assets), self.cache_path) )
      self.refcount += 1
      return self

  def __exit__(self, exc_type, exc_val, exc_tb):
    with self.ctx_lock :
      self.refcount -= 1
      if( self.cache_enabled and self.refcount==0 ):
        with open( self.cache_path, 'wb') as f:
          pickle.dump(self.assets, f)
          info( "MF_session : saved %s assets to %s" % (len(self.assets), self.cache_path) )
        info( "MF_Session : %d cache hits / %d cache miss" % (self._hitcount, self._misscount))
      return False  # do not suppress exceptions

  def set_auth( self, token ):
    with self.auth_lock:
      self.auth= MyAuth( token )  

  def get( self, url, from_cache=True, **kwargs ):
    # prepare the request to build querystring
    req= requests.Request('GET', url, **kwargs )
    full_url = self.prepare_request( req ).url
    
    if( self.cache_enabled and from_cache and full_url in self.assets.keys() and self.auth ):
      self._hitcount += 1
    else:
      self.assets[full_url] = super().get( url, **kwargs ).text
      self._misscount += 1
    return self.assets[full_url]


ztime_lock = Lock()
def parse_ztime(ztime):
  with ztime_lock:
    m = ZTIME_REGEX.match(ztime)
    # convert to integers
    m = { k:int(v) for k,v in m.groupdict().items() }  
    return datetime(  year=m['year'], month=m['month'], day=m['day'], 
                      hour=m['hour'], minute=m['min'], second=m['sec'],
                      tzinfo=timezone.utc )

writedir_lock = Lock()
def write_to_dir(data, filename, dir):
  with writedir_lock:
    # create directory if necessary
    if( not os.path.isdir( dir ) ):
      os.makedirs( dir, exist_ok=True )
    path = os.path.join( dir, filename )
    # write th data
    with open( path, 'w' ) as f:
      f.write( data )

logger_rlock = RLock()
def truncate_msg( msg, nb=125 ):
  with logger_rlock:
    ret = msg
    if( len(msg)>nb ) :
      ret = msg[0:nb]+' ...'
    return ret

def debug( msg, *args, **kwargs ):
  with logger_rlock:
    logging.getLogger(__name__).debug( truncate_msg(msg), *args, **kwargs)
  
def info( msg, *args, **kwargs ):
  with logger_rlock:
    logging.getLogger(__name__).info( truncate_msg(msg), *args, **kwargs)

def error( msg, *args, **kwargs ):
  with logger_rlock:
    logging.getLogger(__name__).error( truncate_msg(msg), *args, **kwargs)

#############################################################################
# Application 
#############################################################################
class MainApp():

  def __init__(self):
    self.args = self.parse_cmdline()
    self.configure_logs()
    logging.info( str(self.args) )
    os.chdir( self.get_script_dir() )
    debug("Working dir : '%s'"% os.getcwd() )

    # pour avoir les dates dans la locale de la plateforme ( français )
    locale.setlocale( locale.LC_ALL, 'fr_FR.UTF-8' )
   
  def parse_cmdline(self):
      parser = argparse.ArgumentParser( description="La meteo de Le_Vincent" )
      parser.add_argument( '-r', '--refresh-assets', action='store_false', dest='cache_assets',
        help="Ignore le cache des données fixes (fonds de carte, pictos, etc...)")
      parser.add_argument( '-o', '--old-prevs', action='store_true', dest='cache_data',
        help="Ne telecharge pas de nouvelles prévisions, utilise seulement le cache")
      parser.add_argument( '-w', '--workers', action='store', default=4, type=int,
        help='Nombre de connexions simultanées' )  
      parser.add_argument( '-t', '--test', action='store_true',
        help="Limite à 1 seule région")
      parser.add_argument( '-v', '--verbose', action='count', default=0, 
        help='Augmente le niveau de verbosité' )
      parser.add_argument( '-p', '--prod', action='store_true',
        help='Mode production - execution periodique' )  
      return parser.parse_args()

# https://stackoverflow.com/a/22881871
  def get_script_dir( self, follow_symlinks=True):
    if getattr(sys, 'frozen', False): # py2exe, PyInstaller, cx_Freeze
        path = os.path.abspath(sys.executable)
    else:
        path = inspect.getabsfile(self.get_script_dir)
    if follow_symlinks:
        path = os.path.realpath(path)
    return os.path.dirname(path)

  def configure_logs(self):
#    logging.basicConfig( level = logging.DEBUG, format='%(levelname)s: %(message)s' ) 
    format_str="%(asctime)s %(levelname)s %(threadName)s %(name)s %(message)s"
    logging.basicConfig( level = logging.DEBUG, format=format_str )
    levels = {
      0: logging.WARNING,
      1: logging.INFO,
      2: logging.DEBUG
    }
    l=min(len(levels)-1, self.args.verbose)
    logging.getLogger().setLevel( levels[ l ])

  def run_once(self):
    c = Mf_Crawler( cache_data = self.args.cache_data, 
                    cache_assets = self.args.cache_assets,
                    test_mode = self.args.test)
    c.run_threads(self.args.workers)

  def run( self, hours=4 ):
    # mode production: catch/ignore all exception and run every 4 hours
    if( self.args.prod):
      while( True ):
        try:
          # create a separate process to properly release RAM between runs
          p = Process( target= MainApp().run_once )
          p.start()
          p.join()
          p.close()
        except Exception as e:
          error( str(e) )
        time_utils.sleep( hours*3600)
    # mode developpement : run_once
    else:
      self.run_once()

###############################################################################
# Main()
###############################################################################
if( __name__ == "__main__"):
  app = MainApp()
  app.run()
