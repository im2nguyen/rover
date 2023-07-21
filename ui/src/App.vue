<template>
  <div id="app">
    <main-nav @saveGraph="saveGraph" @resetZoom="resetZoom" />
    <div class="row">
      <div class="col col-4-lg">
        <fieldset>
          <legend>Legend</legend>
          <b>Instructions</b>
          <hr />
          <p>
            Click or hover on node to isolate that node's connections. Click on
            the light purple background to unselect.
          </p>
          <p>
            All resources that the node depends on are represented by a solid
            line. All resources that depend on the node are represented by a
            dashed line.
          </p>
          <hr />
          <b>Resource</b>
          <hr />
          <div class="node create">Resource - Create</div>
          <div class="node delete">Resource - Delete</div>
          <div class="node replace">Resource - Replace</div>
          <div class="node update">Resource - Update</div>
          <div class="node no-op">Resource - No Operation</div>
          <hr />
          <b>Other items</b>
          <hr />
          <div class="node variable">Variable</div>
          <div class="node output">Output</div>
          <div class="node data">Data</div>
          <div class="node module">Module</div>
          <div class="node locals">Local</div>
          <hr />
        </fieldset>
        <resource-detail :resourceID="resourceID" />
      </div>
      <div class="col col-8-lg">
        <graph
          ref="filegraph"
          :displayGraph="displayGraph"
          v-on:getNode="selectResource"
        />
        <explorer @selectResource="selectResource" />
      </div>
    </div>
  </div>
</template>

<script>
import MainNav from "@/components/MainNav.vue";
import ResourceDetail from "@/components/ResourceDetail.vue";
import Graph from "@/components/Graph/Graph.vue";
// import SampleGraph from "@/assets/eks-graph.json";
import Explorer from "@/components/Explorer.vue";

export default {
  name: "App",
  metaInfo: {
    title: "Rover | Terraform Visualization",
  },
  components: {
    MainNav,
    Graph,
    Explorer,
    ResourceDetail,
  },
  data() {
    return {
      displayGraph: true,
      resourceID: "",
    };
  },
  methods: {
    saveGraph() {
      // this.displayGraph = displayGraph;
      this.$refs.filegraph.saveGraph();
    },
    selectResource(resourceID) {
      this.resourceID = resourceID;
    },
    resetZoom() {
      this.$refs.filegraph.resetZoom();
    },
  },
};
</script>

<style scoped>
#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  /* text-align: center; */
  margin: 0 auto;
  margin-top: 60px;
  width: 90%;
}

.node {
  display: inline-block;
  margin: 0 1%;
  width: 48%;
  font-size: 0.9em;
}

.module {
  border: 5px solid #8450ba;
  color: #8450ba;
}
</style>
