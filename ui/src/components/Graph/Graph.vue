<template>
  <transition name="graph">
    <fieldset>
      <legend>Graph</legend>
      <cytoscape ref="cy" :config="config" :preConfig="preConfig"></cytoscape>
    </fieldset>
  </transition>
</template>

<script>
import { saveAs } from "file-saver";
import klay from "cytoscape-klay";
import nodeHtmlLabel from "cytoscape-node-html-label";
import axios from "axios";

const config = {
  autounselectify: true,
  style: [
    {
      selector: "node",
      style: {
        label: "data(label)",
        width: "500px",
        "font-family": "Avenir, Helvetica, Arial, sans-serif",
        "font-size": "2em",
      },
    },
    {
      selector: "edge",
      css: {
        "curve-style": "taxi",
        "line-fill": "linear-gradient",
        "line-gradient-stop-colors": "data(gradient)",
        "line-dash-offset": 24,
        width: 10,
      },
    },
    {
      selector: ".basename",
      style: {
        padding: "200px",
        "text-margin-y": 75,
        "font-weight": "bold",
        shape: "roundrectangle",
        "min-height": "400px",
        "border-width": 2,
        "border-color": "white",
        "background-color": "#f4ecff",
      },
    },
    {
      selector: ".fname",
      style: {
        padding: "100px",
        "text-margin-y": 75,
        "font-weight": "bold",
        shape: "roundrectangle",
        "border-width": 1,
        "border-color": "lightgrey",
        "background-color": "white",
      },
    },
    {
      selector: ".provider",
      style: {
        "text-valign": "center",
        "text-halign": "center",
        padding: "1em",
        shape: "roundrectangle",
        "border-width": 0,
        color: "white",
        "background-color": "black",
      },
    },
    {
      selector: ".module",
      style: {
        padding: "100px",
        "font-weight": "bold",
        "text-margin-y": 60,
        shape: "roundrectangle",
        color: "#8450ba",
        "border-width": 10,
        "border-color": "#8450ba",
        "background-color": "white",
      },
    },
    {
      selector: ".data-type",
      style: {
        padding: "10%",
        width: "label",
        "font-weight": "bold",
        "text-background-color": "white",
        "text-background-opacity": 1,
        "text-background-padding": "2em",
        "text-margin-y": 15,
        shape: "roundrectangle",
        "border-width": "5px",
        "border-color": "black",
        "background-color": "white",
        // "text-background-color": "data(parentColor)",
        // "background-color": "data(parentColor)",
      },
    },
    {
      selector: ".data-name",
      css: {
        "background-color": "#ffecec",
        color: "black",
        "font-weight": "bold",
        "text-valign": "center",
        "text-halign": "center",
        padding: "1.5em",
        shape: "roundrectangle",
        "border-opacity": 1,
        "border-width": 5,
        "border-color": "#dc477d",
        label: "data(label)",
      },
    },
    {
      selector: ".output",
      css: {
        "background-color": "#fff7e0",
        color: "black",
        "font-weight": "bold",
        "text-valign": "center",
        "text-halign": "center",
        padding: "1.5em",
        shape: "roundrectangle",
        "border-opacity": 1,
        "border-width": 5,
        "border-color": "#ffc107",
        label: "data(label)",
      },
    },
    {
      selector: ".variable",
      css: {
        "background-color": "#e1f0ff",
        color: "black",
        "font-weight": "bold",
        "text-valign": "center",
        "text-halign": "center",
        padding: "1.5em",
        shape: "roundrectangle",
        "border-opacity": 1,
        "border-width": 5,
        "border-color": "#1d7ada",
        label: "data(label)",
      },
    },
    {
      selector: ".locals",
      css: {
        "background-color": "black",
        color: "white",
        "font-weight": "bold",
        "text-valign": "center",
        "text-halign": "center",
        padding: "1.5em",
        shape: "roundrectangle",
        "border-opacity": 1,
        "border-width": 5,
        "border-color": "black",
        label: "data(label)",
      },
    },
    {
      selector: ".resource-type",
      style: {
        padding: "10%",
        width: "label",
        "font-weight": "bold",
        "text-background-color": "white",
        "text-background-opacity": 1,
        "text-background-padding": "2em",
        "text-margin-y": 15,
        shape: "roundrectangle",
        "border-width": "5px",
        "border-color": "black",
        "background-color": "white",
        // "text-background-color": "data(parentColor)",
        // "background-color": "data(parentColor)",
      },
    },
    {
      selector: ".resource-parent",
      style: {
        padding: "10%",
        width: "label",
        "font-weight": "bold",
        "text-background-color": "white",
        "text-background-opacity": 1,
        "text-background-padding": "2em",
        "text-margin-y": 15,
        shape: "roundrectangle",
        "border-width": "5px",
        "border-color": "black",
        "background-color": "white",
        // "text-background-color": "data(parentColor)",
        // "background-color": "data(parentColor)",
      },
    },
    {
      selector: ".resource-name",
      css: {
        "text-valign": "center",
        "text-halign": "center",
        padding: "1.5em",
        shape: "roundrectangle",
        "border-opacity": 0,
        color: "white",
        "background-color": "#8450ba",
        "text-wrap": "ellipsis",
        "text-max-width": 500,
      },
    },
    {
      selector: ".create",
      css: {
        "background-color": "#28a745",
        color: "white",
        "font-weight": "bold",
      },
    },
    {
      selector: ".destroy",
      css: {
        "background-color": "#e40707",
        color: "white",
        "font-weight": "bold",
      },
    },
    {
      selector: ".update",
      css: {
        "background-color": "#1d7ada",
        color: "white",
        "font-weight": "bold",
      },
    },
    {
      selector: ".replace",
      css: {
        "background-color": "#ffc107",
        color: "black",
        "font-weight": "bold",
      },
    },
    {
      selector: ".no-op",
      css: {
        color: "black",
        "border-opacity": 1,
        "font-weight": "bold",
        "border-width": "5px",
        "border-color": "lightgray",
        "background-color": "white",
      },
    },
    {
      selector: ".invisible",
      css: {
        opacity: "0",
      },
    },
    {
      selector: ".semitransp",
      css: {
        opacity: "0.4",
      },
    },
    {
      selector: "edge.semitransp",
      css: {
        opacity: "0",
      },
    },
    {
      selector: ".visible",
      css: {
        opacity: "1",
      },
    },
    {
      selector: ".dashed",
      css: {
        "line-style": "dashed",
        "line-dash-pattern": [20, 20],
      },
    },
  ],
};

export default {
  name: "Graph",
  data() {
    return {
      selectedNode: "",
      config,
      graph: {},
    };
  },
  methods: {
    preConfig(cy) {
      cy.use(klay);

      // Only load nodeHtmlLabel once
      if (typeof cy("core", "nodeHtmlLabel") !== "function") {
        cy.use(nodeHtmlLabel);
      }
    },
    // async afterCreated() {
    //   this.renderGraph();
    // },
    renderGraph: function () {
      let vm = this;
      let cy = this.$refs.cy.instance;
      let el = cy.elements();
      const nodesNames = this.graph.nodes.map(
        x => x.data.id
      )

      // Reset graph
      cy.remove(el);

      // Add nodes
      this.graph.nodes.forEach((n) => {
        cy.add(n);
      });

      // Add edges
      this.graph.edges.forEach((n) => {
        if (n.data.id.includes("-variable") || n.data.id.includes("-output")) {
          return;
        }

        // Browse node ancestors if target is not found in nodes
        // e.g: resource.test.attribute will not be in nodes, as attributes are not exported
        // this ensures that the dependency will be made on resource.test instead
        let name = n.data.target
        while (!nodesNames.includes(name)) {
          name = name.split('.')
          if (name.length < 2) {
            console.warn("edge target", n.data.target, "not found in nodes")
            return
          }
          name.pop()
          name = name.join('.')
        }
        n.data.target = name

        // Add edge to the final graph
        cy.add(n);
      });

      // cy.nodeHtmlLabel([
      //   {
      //     query: ".resource-name",
      //     tpl: function (data) {
      //       return `<div class="node ${data.change ? data.change : ""}">${
      //         data.label
      //       }</div>`;
      //     },
      //   },
      //   {
      //     query: ".data-name",
      //     tpl: function (data) {
      //       return `<div class="node data ${data.change ? data.change : ""}">${
      //         data.label
      //       }</div>`;
      //     },
      //   },
      //   {
      //     query: ".variable",
      //     tpl: function (data) {
      //       return `<div class="node variable">${data.label}</div>`;
      //     },
      //   },
      //   {
      //     query: ".output",
      //     tpl: function (data) {
      //       return `<div class="node output">${data.label}</div>`;
      //     },
      //   },
      //   {
      //     query: ".locals",
      //     tpl: function (data) {
      //       return `<div class="node locals">${data.label}</div>`;
      //     },
      //   },
      // ]);

      this.runLayouts();

      // Add click event
      cy.on("click", "node", function (event) {
        var n = event.target;

        let node = { id: n.data().id, in: [], out: [] };
        const ce = n.connectedEdges();
        for (let i = 0; i < ce.length; i += 1) {
          let ed = ce[i].data();
          if (n.data().id === ed.source) {
            node.out.push(ed.target);
          } else {
            node.in.push(ed.source);
          }
        }

        // When click on resource group
        let rg = node.id.split("/")[1];
        if (rg) {
          if (rg.endsWith(".tf")) {
            return;
          }
        }

        // When click on directory
        if (["basename", "fname"].includes(n.data().type)) {
          vm.selectedNode = "";
          vm.unhighlightNodePaths(n);
          return;
        } else {
          vm.selectedNode = node.id;
          vm.highlightNodePaths(n);
        }

        const na = n.ancestors();
        let nodeID = [];

        for (let i = na.length - 1; i > 0; i--) {
          nodeID.push(na[i].id());
        }

        nodeID.push(n.id());

        vm.$emit("getNode", nodeID.join("/"));
      });

      // Add hover event
      cy.on("mouseover", "node", function (event) {
        let node = event.target;
        if (!vm.selectedNode) {
          vm.highlightNodePaths(node);
        }
      });
      cy.on("mouseout", "node", function (event) {
        var node = event.target;
        if (!vm.selectedNode) {
          vm.unhighlightNodePaths(node);
        }
      });
    },
    highlightNodePaths: function (node) {
      let cy = this.$refs.cy.instance;
      if (
        !["basename", "fname"].includes(node.data().type) &&
        (!node.isParent() || node.data().type === "module")
      ) {
        // make everything but current node and parent transparent
        cy.elements()
          .difference(node.outgoers().union(node.incomers()))
          .filter(function (e) {
            if (!["basename", "fname"].includes(e.data().type)) {
              return e;
            }
          })
          .not(node)
          .not(node.parent())
          .not(node.parent().parent())
          .addClass("semitransp");

        node
          .neighborhood()
          .union(node.neighborhood().parent())
          .addClass("visible");

        node.incomers().addClass("dashed");
      }
    },
    unhighlightNodePaths: function (node) {
      let cy = this.$refs.cy.instance;
      if (!node.data().type.includes[("basename", "fname")]) {
        cy.elements()
          .removeClass("semitransp")
          .removeClass("visible")
          .removeClass("dashed");
      }
    },
    saveGraph: function () {
      let cy = this.$refs.cy.instance;
      saveAs(cy.png({ full: true }), `rover.png`);
    },
    runLayouts: function () {
      let cy = this.$refs.cy.instance;

      cy.layout({
        name: "klay",
        nodeDimensionsIncludeLabels: true,
        klay: {
          direction: "RIGHT",
          borderSpacing: 100,
          spacing: 30,
        },
      }).run();
    },
  },
  mounted() {
    // if graph.js file is present (standalone mode)
    // eslint-disable-next-line no-undef
    if (typeof graph !== "undefined") {
      // eslint-disable-next-line no-undef
      this.graph = graph;
      this.renderGraph();
    } else {
      axios.get("http://localhost:9000/api/graph").then((response) => {
        this.graph = response.data;
        this.renderGraph();
      });
    }
  },
};
</script>

<style>
#cytoscape-div {
  height: 1000px !important;
  background-color: #f8f8f8 !important;
}

.node {
  width: 14em;
  font-size: 2em;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  text-align: center;
  padding: 0.5em 0.5em;
  border-radius: 0.25em;
  background-color: white;
  color: black;
  font-weight: bold;
  cursor: pointer;
  border: 5px solid lightgray;
}

.node:hover {
  transform: scale(1.02);
}

.resource-type {
  width: 20em;
  font-size: 2em;
  height: 100%;
}

.create {
  background-color: #28a745;
  color: white;
  font-weight: bold;
  border: 0;
}

.destroy {
  /* background-color: #ffe9e9;
  border: 5px solid #e40707; */
  background-color: #e40707;
  color: white;
  font-weight: bold;
  border: 0;
}

.update {
  /* background-color: #e1f0ff;
  border: 5px solid #1d7ada; */
  background-color: #1d7ada;
  color: white;
  font-weight: bold;
  border: 0;
}

.replace {
  /* background-color: #fff7e0;
  border: 5px solid #ffc107; */
  background-color: #ffc107;
  color: black;
  font-weight: bold;
  border: 0;
}

.output {
  background-color: #fff7e0;
  border: 5px solid #ffc107;
  color: black;
  font-weight: bold;
}

.variable {
  background-color: #e1f0ff;
  border: 5px solid #1d7ada;
  color: black;
  font-weight: bold;
}

.data {
  background-color: #ffecec;
  border: 5px solid #dc477d;
  color: black;
  font-weight: bold;
}

.locals {
  background-color: black;
  color: white;
  font-weight: bold;
  border: 0;
}
</style>


<style scoped>
fieldset {
  margin-bottom: 2em;
}

.graph-enter-active,
.graph-leave-active,
.graph-enter-active legend,
.graph-leave-active legend {
  transition: all 0.2s ease;
  overflow: hidden;
}

.graph-enter,
.graph-leave-to,
.graph-enter legend,
.graph-leave-to legend {
  height: 0;
  padding: 0;
  margin: 0;
  opacity: 0;
}
</style>
