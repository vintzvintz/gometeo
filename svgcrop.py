import xml.etree.ElementTree as ET


class TreeBuilderWithDoctype ( ET.TreeBuilder):
  """ add DOCTYPE extraction feature to the vanilla Python TreeBuilder/Parser
  """
  _doctype = ""
  def doctype(self, name, pubid, system):
    self._doctype = '<!DOCTYPE %s PUBLIC "%s" "%s">' % (name, pubid, system) 


class SvgCropper :
  """
  Parse a SVG and resize by cropping
  Modify some attributes in the root <svg> element : viewBox, height and width
  """
  _doctype = ""
  _root    = None

  def __init__(self, svg ):
    tb = TreeBuilderWithDoctype()
    parser = ET.XMLParser( target=tb )
    self._root = ET.fromstring( svg, parser=parser )
    self._doctype = tb._doctype

  def __str__(self):
    return '<?xml version="1.0" encoding="utf-8"?>' +      \
            self._doctype +                                \
            ET.tostring( self._root, encoding='unicode') 

  def crop(self, crop_O, crop_E, crop_N, crop_S ):
    # attribut viewBox : [min_x min_y width height]
    viewbox = [ int(x) for x in self._root.attrib['viewBox'].split() ]

    # vérifie la cohérence entre les dimensions du viewbox et les dimensions de l'image en pixels
    # remove 'px' in height and width 
    if ( viewbox[3] != int (self._root.attrib['height'][:-2]) or \
         viewbox[2] != int (self._root.attrib['width'][:-2])) :
      raise Exception( "SvgCropper : Incohérence ddans les dimensions" )

    vb_crop = [ viewbox[0]+crop_O*viewbox[2],  \
                viewbox[1]+crop_N*viewbox[3],  \
                viewbox[2]-(crop_O+crop_E)*viewbox[2],   \
                viewbox[3]-(crop_N+crop_S)*viewbox[3] ]  

    self._root.set( 'width',  '%dpx' % vb_crop[2] )
    self._root.set( 'height', '%dpx' % vb_crop[3] )
    self._root.set( 'viewBox', ' '.join(["%d"%n for n in vb_crop]) )


if( __name__ == '__main__' ):
  with open( '/tmp/regin10.svg') as f:
    svg = SvgCropper( f.read() )
  svg.crop(150,25,60,60)
  with open('/tmp/cropped.svg', 'w', encoding='utf-8') as f:
    f.write( str(svg) )





