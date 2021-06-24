<template>
  <div class="file" v-if="fileName">
    <div class="row" @click="showChildren = !showChildren">
      <img :src="expandIcons[expandIcon]" class="file-expand-icon" />
      <div class="col-11 file-name">
        <strong class="text-lowercase">{{ fileName }}</strong>
      </div>
    </div>
    <template v-for="resource in sortedResources">
      <transition-group name="resources" :key="resource[0]">
        <resource-card
          :key="resource[0]"
          :id="resource[0]"
          :content="resource[1]"
          :isChild="false"
          v-if="showChildren"
          :handle-click="selectResource"
        />
      </transition-group>
    </template>
  </div>
</template>

<script>
import ResourceCard from "@/components/ResourceCard.vue";

export default {
  name: "File",
  components: {
    ResourceCard,
  },
  props: {
    fileName: String,
    resources: Object,
  },
  data() {
    return {
      showChildren: true,
      expandIcons: {
        expand: require("@/assets/icons/arrow-down-circle.svg"),
        collapse: require("@/assets/icons/arrow-up-circle.svg"),
      },
    };
  },
  methods: {
    selectResource(resourceID) {
      this.$emit("selectResource", `${this.fileName}/${resourceID}`);
    },
  },
  computed: {
    expandIcon() {
      return this.showChildren ? "collapse" : "expand";
    },
    sortedResources() {
      // Sort by line number
      const sorted = Object.entries(this.resources).sort(
        (x, y) => x[1].line - y[1].line
      );

      // Sort by name
      // const sorted = Object.entries(this.resources).sort(
      //   (x, y) => x[1].name - y[1].name
      // );

      // Sort by type
      // const sorted = Object.entries(this.resources).sort(
      //   (x, y) => x[1].type - y[1].type
      // );

      // console.log(sorted);
      return sorted;
    },
  },
};
</script>

<style scoped>
.file {
  margin-bottom: 1em;
}

.file-name {
  margin-bottom: 0;
  margin-top: 0.25em;
}

.file-name:hover {
  cursor: pointer;
}

.resources-enter-active,
.resources-leave-active {
  transition: all 0.2s ease;
  overflow: hidden;
}

.resources-enter, .resources-leave-to /* .fade-leave-active below version 2.1.8 */ {
  height: 0;
  padding: 0;
  margin: 0;
  opacity: 0;
}

.file-expand-icon {
  width: 1em;
  padding-top: 0.1em;
  margin-left: 1.4em;
}
</style>
