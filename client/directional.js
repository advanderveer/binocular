
// get the data
d3.json('/containers', function(error, craw) {
d3.json('/logs', function(error, raw) {

var anonymous = {
    ID: "000000000000000000",
    Names: ["/anonymous"],
    Image: ["scratch"]
}

var lowValue = 1;
var hiValue = 1;
var maxArrowWidth = 1.4;
var minArrowWidth = 0.5;

var containers = {}
var nodes = {};
var links = [];
var width = window.innerWidth,
    height = window.innerHeight;

craw.forEach(function(c){
    containers[c.ID] = c
})

function upsertLink(from, to){
  var existing = links.filter(function(l){
    return l.source === from && l.target === to
  })

  if(existing.length == 0) {
    links.push({source: from, target: to, value: 1})
  } else {
    //increment value
    existing.forEach(function(ex){
      ex.value++
    })
  }
}



//raw to links
raw.forEach(function(l){
  upsertLink(l.From, l.To)
})

// Compute the distinct nodes from the links.
links.forEach(function(link) {    
    link.source = nodes[link.source] || 
        (nodes[link.source] = {name: link.source});
    link.target = nodes[link.target] || 
        (nodes[link.target] = {name: link.target});
    link.value = +link.value;    
});

//create index for neighbouring
var linkedByIndex = {};
links.forEach(function(d) {    
    linkedByIndex[d.source.name + "," + d.target.name] = 1;
});

//find lowest and highest link value 
links.forEach(function(d){
    if(d.value < lowValue) {
        lowValue = d.value
    }

    if(d.value > hiValue) {
        hiValue = d.value
    }
})

function isConnected(a, b) {    
    res = linkedByIndex[a.name + "," + b.name]  || a.name == b.name;
    return res
}

var force = d3.layout.force()
    .nodes(d3.values(nodes))
    .links(links)
    .size([width, height])
    .linkDistance(300)
    .charge(-2000)
    .on("tick", tick)
    .start();

var svg = d3.select("body").append("svg")
    .attr("width", width)
    .attr("height", height);

function resize() {
  width = window.innerWidth, height = window.innerHeight;
  svg.attr("width", width).attr("height", height);
  force.size([width, height]).resume();
  voronoi.clipExtent([[-10, -10], [width+10, height+10]]);
}

d3.select(window).on("resize", resize);

// build the arrow.
svg.append("svg:defs").selectAll("marker")
    .data(["end"])      // Different link/path types can be defined here
  .enter().append("svg:marker")    // This section adds in the arrows
    .attr("id", String)
    .attr("viewBox", "0 -5 10 10")
    .attr("refX", 15)
    .attr("refY", -1.5)
    .attr("markerWidth", 6)
    .attr("markerHeight", 6)
    .attr("orient", "auto")
  .append("svg:path")
    .attr("d", "M0,-5L10,0L0,5");

// add the links and the arrows
var path = svg.append("svg:g").selectAll("path")
    .data(force.links())
  .enter().append("svg:path")
    .attr("class", "link")
    .attr("marker-end", "url(#end)")
    .style("stroke-width", function(d) {
        var scale = (hiValue - lowValue)
        var rel = (d.value - lowValue)
        
        return minArrowWidth+(rel*(maxArrowWidth - minArrowWidth)); 
    });

// define the nodes
var node = svg.selectAll(".node")
    .data(force.nodes())
  .enter().append("g")
    .attr("class", "node")
    .call(force.drag)
    .on("mouseover", fade(.1)).on("mouseout", fade(1));;

// add the nodes
node.append("circle")
    .attr("r", 5);

//selection circle
node.append('circle')
    .attr('r', 30)
    .attr('fill', "#EFEFEF")
    .attr('fill-opacity', 0.5);


// add the container name(s)
node.append("text")
    .attr("x", 12)
    .attr("class", "label-cname")
    .attr("dy", "-.8em")
    .text(function(d) { 
        var c = containers[d.name]
        if(!c) {
            c = anonymous
        }
        return c.Names.join(", "); 
    });

// add the container id
node.append("text")
    .attr("x", 12)
    .attr("class", "label-cid")
    .attr("dy", ".35em")
    .text(function(d) { 
        return d.name.substring(0,8); 
    });

// add the container image name
node.append("text")
    .attr("x", 12)
    .attr("class", "label-iid")
    .attr("dy", "1.5em")
    .text(function(d) { 
        var c = containers[d.name]
        if(!c) {
            c = anonymous
        }
        return c.Image
    });


//intersection rendering
var voronoi = d3.geom.voronoi()
    .x(function(d) { return d.x; })
    .y(function(d) { return d.y; })
    .clipExtent([[-10, -10], [width+10, height+10]]);

function recenterVoronoi(nodes) {
    var shapes = [];
    voronoi(nodes).forEach(function(d) {
        if ( !d.length ) return;
        var n = [];
        d.forEach(function(c){
            n.push([ c[0] - d.point.x, c[1] - d.point.y ]);
        });
        n.point = d.point;
        shapes.push(n);
    });
    return shapes;
}

//fade others on 
function fade(opacity) {
    return function(d) {
        node.style("stroke-opacity", function(o) {
            var isc = isConnected(d, o)
            thisOpacity = isc ? 1 : opacity;
            this.setAttribute('fill-opacity', thisOpacity);
            return thisOpacity;
        });

        path.style("opacity", function(o) {
            return o.source === d ? 1 : opacity;
        });
    };
}

// add the curvy lines
function tick() {
    path.attr("d", function(d) {
        var dx = d.target.x - d.source.x,
            dy = d.target.y - d.source.y,
            dr = Math.sqrt(dx * dx + dy * dy);
        return "M" + 
            d.source.x + "," + 
            d.source.y + "A" + 
            dr + "," + dr + " 0 0,1 " + 
            d.target.x + "," + 
            d.target.y;
    });

    node
        .attr('clip-path', function(d) { return 'url(#clip-'+d.index+')'; })
        .attr("transform", function(d) { 
        return "translate(" + d.x + "," + d.y + ")"; });
}

})});