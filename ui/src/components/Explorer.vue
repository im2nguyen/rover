<template>
  <fieldset>
    <legend>Resources</legend>
    <!-- {{ overview }} -->
    <File />

    <div v-for="(properties, fileName) in map.root" :key="fileName">
      <File
        :fileName="fileName"
        :resources="properties.children"
        @selectResource="selectResource"
      />
    </div>
  </fieldset>
</template>

<script>
import File from "@/components/File.vue";
import axios from "axios";

export default {
  name: "Explorer",
  components: {
    File,
  },
  data() {
    return {
      map: {},
    };
  },
  methods: {
    selectResource(resourceID) {
      this.$emit("selectResource", resourceID);
    },
  },
  mounted() {
    // if map.js file is present (standalone mode)
    // eslint-disable-next-line no-undef
    if (typeof map !== "undefined") {
      // eslint-disable-next-line no-undef
      this.map = map;
    } else {
      axios.get(`/api/map`).then((response) => {
        this.map = response.data;
        //console.log(this.map);
      });
    }
  },
};
</script>

<style scoped>
fieldset {
  margin-bottom: 2em;
  /* background-color: #292a34; */
}
</style>
