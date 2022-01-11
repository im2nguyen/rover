<template>
  <div
    class="card resource-main"
    :class="[
      isChild ? 'child' : '',
      `resource-card ${content.type}`,
      content.change_action != null ? content.change_action : '',
      content.change_action != null ? '' : 'resource-type-card',
    ]"
  >
    <div class="row" @click="handleClick(id)">
      <div class="col col-6 resource-col">
        <!-- Multiple Resources -->
        <p
          class="is-small resource-action"
          @click="showChildren = !showChildren"
        >
          <img
            :src="expandIcons[expandIcon]"
            class="multi-tag resource-action-icon"
          />
        </p>
        <!-- {{ content }} -->
        <!-- Resource Action -->
        <!-- <p class="is-small resource-action">
          <img
            :src="resourceChangeIcons[content.change_action]"
            class="resource-action-icon"
          />
        </p> -->
        <!-- Resource Name -->
        <p class="resource-name">
          {{ content.name }}
        </p>
      </div>
      <div class="col col-4">
        <!-- Provider Icons -->
        <template v-if="resourceProvider">
          <img
            class="provider-icon"
            :src="providerIcon[resourceProvider]"
            v-if="providerIcon[resourceProvider]"
          />
          <span class="tag is-small provider-icon-tag" v-else>
            {{ resourceProvider[0] }}
          </span>
        </template>
        <p class="provider-resource-name">
          {{ resourceProvider ? `${resourceProvider}.` : ""
          }}{{ content.resource_type ? content.resource_type : "" }}
        </p>
      </div>
      <div class="col col-2 text-right" v-if="content.line">
        Line: # <span class="line-number">{{ content.line }}</span>
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
          :handle-click="handleClick"
        />
      </transition-group>
    </template>
  </div>
</template>

<script>
export default {
  name: "ResourceCard",
  props: {
    id: String,
    content: Object,
    isChild: Boolean,
    handleClick: Function,
  },
  data() {
    return {
      showChildren: false,
      providerIcon: {
        aws: require("@/assets/provider-icons/aws.png"),
        azure: require("@/assets/provider-icons/azure.png"),
        gcp: require("@/assets/provider-icons/gcp.png"),
        helm: require("@/assets/provider-icons/helm.png"),
        kubernetes: require("@/assets/provider-icons/kubernetes.png"),
      },
      resourceChangeIcons: {
        create: require("@/assets/resource-icons/plus.svg"),
        read: null,
        "no-op": null,
        update: require("@/assets/resource-icons/alert-triangle.svg"),
        delete: require("@/assets/resource-icons/minus.svg"),
        replace: require("@/assets/resource-icons/refresh-cw.svg"),
      },
      expandIcons: {
        null: null,
        expand: require("@/assets/icons/arrow-down-circle.svg"),
        collapse: require("@/assets/icons/arrow-up-circle.svg"),
      },
    };
  },
  methods: {
    // selectChildResource(resourceID) {
    //   console.log(resourceID);
    //   if (resourceID) {
    //     this.$emit("selectResource", resourceID);
    //     return;
    //   }
    //   this.$emit("selectResource", this.id);
    // },
  },
  computed: {
    expandIcon() {
      if (this.content.children) {
        if (this.showChildren) {
          return "collapse";
        }
        return "expand";
      }
      return null;
    },
    sortedResources() {
      // Sort by line number
      if (this.content.children) {

        const sorted = Object.entries(this.content.children).sort(
          (x, y) => x[1].line - y[1].line
        );

        return sorted
      }
      return null;
    },
    resourceProvider() {
      if (this.content.provider) {
        return this.content.provider;
      }

      if (this.content.resource_type) {
        return this.content.resource_type.split("_")[0];
      }

      return null;
    },
  },
};
</script>

<style scoped>
.card {
  margin: 0.5em 0;
  border-radius: 0;
  border-width: 2px;
  font-weight: normal;
}

.tag {
  border: 1px solid var(--color-grey);
}

.card.child {
  margin: 0em -1.3em;
}

.card.child:hover {
  border-width: 2px;
  border-left: 0px solid;
  border-right: 0px solid;
  filter: brightness(0.95);
}

.col {
  margin-bottom: 0;
}

.resource-main:hover {
  cursor: pointer;
  /* filter: brightness(100%); */
  /* background-color: red; */
  /* border-width: 3px; */
  filter: brightness(0.95);
}

.child.resource-main {
  border-left: 1px solid;
  border-right: 1px solid;
}

/* .child.resource-main { */
/* background-color: lavender; */
/* } */

/* .child.resource-main:hover {
  background-color: #d1c7ff;
} */

.dark .resource-main:hover {
  cursor: pointer;
  background-color: #0d032b;
}

.dark .child.resource-main {
  background-color: #1c1c3f;
}

.dark .child.resource-main:hover {
  background-color: #131342 !important;
}

/* .dark .resource-main:hover .resource-action,
.dark .resource-main:hover .resource-type,
.dark .resource-main:hover .provider-icon {
  border-color: black;
  color: black;
} */

.resource-col {
  margin-left: 0.1em;
}

.resource-action {
  float: left;
  margin: 0;
  margin-right: 0.5em;
}

.file-expand-icon {
  width: 1em;
  padding-top: 0.1em;
}

.resource-action-icon {
  width: 1em;
  padding-top: 0.1em;
}

.dark .multi-tag {
  filter: invert(100%);
}

/* .resource-type-card {
  float: left;
  margin: 0;
  margin-right: 1em;
  font-weight: bold;
} */

.resource-name {
  width: 80%;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  float: left;
}

.provider-icon-tag {
  float: left;
  margin: 0 1em 0 0 !important;
  font-weight: bold;
}

.provider-icon {
  float: left;
  width: 1.75em;
  margin: -0.2em 0.5em 0 -0.3em !important;
}

.provider-resource-name {
  width: 85%;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  float: left;
}

.line-number {
  display: inline-block;
  min-width: 2em;
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

.module {
  border: 2px solid #8450ba;
}

.resource-card.create {
  border-color: #28a745;
}

.resource-card.output {
  border-color: #ffc107;
}

.resource-card.delete {
  border-color: #e40707;
}

.resource-card.update {
  border-color: #1d7ada;
}

.resource-card.replace {
  border-color: #ffc107;
}

.resource-type-card {
  margin-top: 0.5em !important;
}
</style>
