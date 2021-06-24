<template>
  <nav class="nav">
    <div class="nav-left">
      <h2>Rover - Terraform Visualizer</h2>
    </div>
    <div class="nav-right">
      <a class="button outline" @click="saveGraph()">Save Graph</a>
      <!-- <a class="button outline" @click="toggleGraph()">{{
        graph ? "Hide Graph" : "Show Graph"
      }}</a> -->
      <!-- <a class="button icon-only clear" @click="switchMode(this)">{{
        colorMode
      }}</a> -->
    </div>
  </nav>
</template>

<script>
export default {
  name: "MainNav",
  data() {
    return {
      colorMode: "☀️",
      graph: true,
    };
  },
  methods: {
    switchMode() {
      const bodyClass = document.body.classList;
      bodyClass.contains("dark")
        ? ((this.colorMode = "☀️"), bodyClass.remove("dark"))
        : ((this.colorMode = "🌙"), bodyClass.add("dark"));

      localStorage.colorMode = this.colorMode;
    },
    // toggleGraph() {
    //   this.graph = !this.graph;

    //   this.$emit("toggleGraph", this.graph);
    // },
    saveGraph() {
      this.$emit("saveGraph", true);
    },
  },
  mounted() {
    // Toggle dark mode
    // if (localStorage.colorMode) {
    //   this.colorMode = localStorage.colorMode;
    // } else {
    //   if (
    //     window.matchMedia &&
    //     window.matchMedia("(prefers-color-scheme: dark)").matches
    //   ) {
    //     this.colorMode = "🌙";
    //   }
    // }
    // localStorage.colorMode = this.colorMode;
    // if (this.colorMode === "☀️") {
    //   document.body.classList.remove("dark");
    // } else {
    //   document.body.classList.add("dark");
    // }
  },
};
</script>