<template>
  <fieldset id="resource-details">
    <legend>Details</legend>
    <div class="resource-detail">
      <div v-if="!resourceID">
        <span>Please select a resource on your right.</span>
      </div>
      <div v-else>
        <dd class="key">{{ primitiveType }}</dd>
        <span
          class="tag is-small resource-action"
          v-if="resourceChange.action"
          >{{ resourceChange.action }}</span
        >
        <dt class="value resource-id">
          {{ resource.id }}
          <button
            class="copy-button"
            @click="copyText(resource.id, 'rid')"
            ref="rid"
          >
            Copy
          </button>
        </dt>

        <!-- <dd class="key">Resource Type</dd>
        <dt class="value">{{ resource.resource_type }}</dt>
c
        <dd class="key">Resource Name</dd>
        <dt class="value">{{ resource.resource_name }}</dt> -->

        <nav class="tabs is-full">
          <a
            @click="selectTab('config')"
            :class="{ active: curTab === 'config' }"
            >Config</a
          >
          <a
            @click="selectTab('current')"
            :class="{ active: curTab === 'current', disabled: hasNoState }"
            >Current State</a
          >
          <a
            @click="selectTab('proposed')"
            :class="{ active: curTab === 'proposed', disabled: hasNoState }"
            >Proposed State</a
          >
        </nav>

        <div class="tab-container" v-if="curTab === 'config'">
          <!-- {{ resourceConfig }} -->
          <span
            v-if="
              resourceConfig.isChild == 'rover-for-each-child-resource-true'
            "
            class="is-child-resource"
            >Please check parent resource</span
          >
          <div v-for="(val, k) in resourceConfig" :key="k" v-else>
            <dd class="key">{{ k }}</dd>
            <dt class="value">
              <span>{{ getConfigValue(val) }}</span>
              <button
                class="copy-button"
                @click="copyText(getConfigValue(val), `${resource.id}-${k}`)"
                :ref="`${resource.id}-${k}`"
              >
                Copy
              </button>
            </dt>
          </div>
        </div>

        <div class="tab-container" v-if="curTab === 'current'">
          <span v-if="resourceChange.before">
            <div v-for="(val, k) in resourceChange.before" :key="k">
              <dd class="key">{{ k }}</dd>
              <dt class="value">
                {{ getBeforeValue(val) }}
                <button
                  class="copy-button"
                  @click="copyText(getBeforeValue(val), `${resource.id}-${k}`)"
                  :ref="`${resource.id}-${k}`"
                >
                  Copy
                </button>
              </dt>
            </div>
          </span>
          <span v-else>Resource doesn't currently exist.</span>
        </div>

        <div class="tab-container" v-if="curTab === 'proposed'">
          <!-- {{ resourceChange }} -->

          <div v-for="(val, k) in resourceChange.after" :key="k">
            <dd class="key">{{ k }}</dd>
            <dt
              class="value"
              v-if="val"
              :class="{ 'unknown-value': val.unknown }"
            >
              {{ val.unknown ? "Value Unknown" : val }}
              <button
                class="copy-button"
                @click="copyText(getBeforeValue(val), `${resource.id}-${k}`)"
                :ref="`${resource.id}-${k}`"
              >
                Copy
              </button>
            </dt>
            <dt class="value" v-else>null</dt>
          </div>
        </div>
      </div>
    </div>
  </fieldset>
</template>

<script>
import axios from "axios";
import copy from "copy-to-clipboard";

export default {
  name: "ResourceDetail",
  props: {
    resourceID: String,
  },
  data() {
    return {
      curTab: "config",
      overview: {},
    };
  },
  methods: {
    selectTab(tab) {
      if (!this.hasNoState) {
        this.curTab = tab;
      }
    },
    copyText(text, ref) {
      copy(text, {
        onCopy: this.updateCopyText(ref),
      });
    },
    updateCopyText(ref) {
      // Use the first element if returns an array
      if (Array.isArray(this.$refs[ref])) {
        this.$refs[ref][0].innerText = "Copied";
        setTimeout(() => {
          this.$refs[ref][0].innerText = "Copy";
        }, 1000);
      } else {
        this.$refs[ref].innerText = "Copied";
        setTimeout(() => {
          this.$refs[ref].innerText = "Copy";
        }, 1000);
      }
    },
    getConfigValue(val) {
      if (val.references) {
        return val.references.join(", ");
      } else if (val.constant_value) {
        return val.constant_value;
      } else {
        return val ? val : "null";
      }
    },
    getBeforeValue(val) {
      return val ? val : "null";
    },
    getAfterValue(val) {
      return val ? val : "null";
    },
    getResourceConfig(resourceID, model, isChild) {
      // console.log(`resourceID: ${resourceID}`);
      // console.log(model);

      // Variables
      if (resourceID.startsWith("var.")) {
        return model.variables[resourceID.replace("var.", "")];
      }
      // Outputs
      if (resourceID.startsWith("output.")) {
        let id = resourceID.replace("output.", "");
        if (model.output[id]) {
          return model.output[id].config;
        }
      }
      // Module
      if (resourceID.startsWith("module.")) {
        if (isChild) {
          let id = resourceID.split(".").slice(2).join(".");

          for (let val of model.module.resources) {
            if (val.address == id) {
              let trc = {};
              if (val.for_each_expression) {
                trc.for_each = val.for_each_expression;
              }
              if (val.count_expression) {
                trc.count = val.count_expression;
              }

              return Object.assign(trc, val.expressions);
            }
          }
        }

        return {
          source: model.source,
          ...model.expressions,
        };
      }
      // Resource
      if (isChild) return { isChild: "rover-for-each-child-resource-true" };
      if (model.resources[resourceID] && model.resources[resourceID].config) {
        let trc = {};
        if (model.resources[resourceID].config.for_each_expression) {
          trc.for_each = model.resources[resourceID].config.for_each_expression;
        }
        if (model.resources[resourceID].config.count_expression) {
          trc.count = model.resources[resourceID].config.count_expression;
        }
        return Object.assign(
          trc,
          model.resources[resourceID].config.expressions
        );
      }

      // Defaults to returning empty object
      return {};
    },
    getResourceChange(resourceID, model, isChild) {
      // console.log(`resourceID: ${resourceID}`);
      // console.log(model);

      let rc = {};

      if (resourceID.startsWith("var.")) {
        return (rc = {});
      }
      if (resourceID.startsWith("output.")) {
        let id = resourceID.replace("output.", "");
        // let id = resourceID;
        if (model.output[id] && model.output[id].change) {
          const c = model.output[id].change;

          if (c.actions) {
            rc.action = c.actions.length > 1 ? "replace" : c.actions[0];
          }
          rc.before = c.before ? c.before : null;
          rc.after = c.after ? c.after : {};

          if (typeof rc.after === "string") {
            rc.after = {
              value: rc.after,
            };
          }

          if (c["after_unknown"]) {
            rc.after["value"] = { unknown: true };
          }

          // console.log(rc);

          return rc;
        }
        return (rc = {});
      }

      if (isChild) {
        if (model.children[resourceID] && model.children[resourceID].change) {
          const c = model.children[resourceID].change;

          // // console.log(c);

          if (c.actions) {
            rc.action = c.actions.length > 1 ? "replace" : c.actions[0];
          }
          rc.before = c.before ? c.before : null;
          rc.after = c.after ? c.after : null;

          if (c["after_unknown"]) {
            for (let k of Object.keys(c["after_unknown"])) {
              rc.after[k] = { unknown: true };
            }
          }

          // // console.log(rc);

          return rc;
        }
        return (rc = {});
      }

      // Resource
      if (model.resources[resourceID] && model.resources[resourceID].change) {
        const c = model.resources[resourceID].change;

        if (c.actions) {
          rc.action = c.actions.length > 1 ? "replace" : c.actions[0];
        }
        rc.before = c.before ? c.before : {};
        rc.after = c.after ? c.after : {};

        if (c["after_unknown"]) {
          Object.keys(c["after_unknown"]).forEach(function (k) {
            rc.after[k] = { unknown: true };
          });
        }
      }

      return rc;
    },
  },
  computed: {
    resource() {
      let resource = "";

      // If no config version...
      if (this.resourceID.startsWith("Resources/")) {
        resource = this.resourceID.split("/").join(".");
      } else {
        resource = this.resourceID.split("/").slice(-2).join(".");
      }

      const rArray = resource.split(".");
      const lastIndex = rArray.length - 1;

      let resourceID = rArray.slice(2).join(".");
      let parentID = rArray.slice(2, 4).join(".").split("[")[0];

      // If no config version..
      if (this.resourceID.startsWith("Resources/")) {
        resourceID = rArray.slice(1).join(".");
        parentID = rArray.slice(1, 4).join(".").split("[")[0];
      }

      if (
        rArray[lastIndex - 1] == "output" &&
        !resourceID.startsWith("output.")
      ) {
        resourceID = `output.${resourceID}`;
      }

      if (rArray[lastIndex - 1] == "local") {
        resourceID = `local.${rArray[lastIndex]}`;
      }

      if (rArray[lastIndex - 1] == "var") {
        resourceID = `var.${rArray[lastIndex]}`;
      }

      // If resourceID is a child only (no . in id)
      if (resourceID.match(/^[\w-]+[[]/g) != null) {
        resourceID = rArray.slice(1).join(".");
        parentID = rArray.slice(1, 4).join(".").split("[")[0];
      }

      return {
        fileName: `${rArray[0]}.${rArray[1]}`,
        id: resourceID,
        parentID: parentID,
        resource_type: rArray[lastIndex - 1],
        resource_name: rArray[lastIndex],
      };
    },
    primitiveType() {
      switch (this.resource.resource_type) {
        case "output":
        case "var":
        case "local":
          return this.resource.resource_type;
        default:
          if (this.resource.id.startsWith("data.")) {
            return "data";
          }
          return "resource";
      }
    },
    isChild() {
      return this.resource.id.match(/^\w+\.[\w-]+[[.]/g) != null;
    },
    hasNoState() {
      return this.resource.id.startsWith("var.");
    },
    resourceConfig() {
      if (this.resource.id === "") {
        return { action: "", before: {} };
      }

      if (!this.isChild) {
        // If it's part of a module
        if (this.resource.id.startsWith("module.")) {
          return this.getResourceConfig(
            this.resource.id,
            this.overview.resources[this.resource.parentID].module_config,
            false
          );
        }
        return this.getResourceConfig(this.resource.id, this.overview, false);
      }

      // If it's part of a module
      if (this.resource.id.startsWith("module.")) {
        return this.getResourceConfig(
          this.resource.id,
          this.overview.resources[this.resource.parentID].module_config,
          true
        );
      }
      return this.getResourceConfig(this.resource.id, this.overview, false);
      // return this.isChild;
    },
    resourceChange() {
      if (this.resource.id === "") {
        return { action: "", before: {} };
      }

      if (!this.isChild) {
        return this.getResourceChange(this.resource.id, this.overview, false);
      }

      if (this.resource.id.startsWith("module.")) {
        return this.getResourceChange(
          this.resource.id,
          this.overview.resources[this.resource.parentID],
          true
        );
      }

      return this.getResourceChange(
        this.resource.id,
        this.overview.resources[this.resource.parentID],
        true
      );
    },
  },
  watch: {
    resourceID: function (newVal) {
      if (newVal.includes("var.")) {
        this.curTab = "config";
      }
    },
  },
  mounted() {
    // if rso.js file is present (standalone mode)
    // eslint-disable-next-line no-undef
    if (typeof rso !== "undefined") {
      // eslint-disable-next-line no-undef
      this.overview = rso;
    } else {
      axios.get(`/api/rso`).then((response) => {
        this.overview = response.data;
      });
    }
  },
};
</script>

<style scoped>
#resource-details {
  position: sticky;
  top: 1em;
  min-width: 0;
  /* background-color: #292a34; */
}
.tab-container {
  max-height: 70vh;
  overflow: scroll;
}
fieldset {
  margin-bottom: 2em;
}
.tabs a:hover {
  cursor: pointer;
}
.resource-detail {
  padding: 1em 0;
}
.tab-container {
  padding: 1em 0;
}
.tabs .disabled:hover {
  cursor: not-allowed;
  border-bottom: 4px solid var(--color-lightGrey);
}

p {
  word-break: break-all;
  white-space: normal;
}

a {
  font-weight: bold;
  border-width: 4px !important;
}

.key {
  font-weight: bold;
  font-size: 0.9em;
  text-transform: uppercase;
  margin: 0;
}

dd {
  display: inline-block;
}

dt.value {
  margin: 0.5em 0 1em 0;
  padding: 0.5em;
  font-size: 1em;
  background-color: #f4ecff;
  color: black;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.resource-id {
  word-wrap: break-word;
  overflow: hidden;
  width: 100%;
}

.resource-action {
  float: right;
}

.is-child-resource {
  display: block;
  text-align: center;
  font-weight: bold;
  font-style: italic;
}

.unknown-value {
  text-align: center;
  font-weight: bold;
  font-style: italic;
}

.copy-button {
  font-size: 0.9em;
  padding: 1rem;
  align-items: flex-end;
  background-color: #8450ba;
  color: white;
  font-weight: bold;
}

.copy-button:hover {
  cursor: pointer;
}
</style>