<template>
  <div ref="content" class="content" :style="{top: axis.y + 'px', left: axis.x + 'px'}" v-if="visibility">
    <a v-for="item in menus" :key="item.name" @click="onMenuClick(item.name)"
       :style="{'text-decoration': item.underline ? 'underline' : 'none'}">
      {{item.caption || item.name}}
    </a>
  </div>
</template>

<script>
export default {
  name: 'ContextMenu',
  props: {
    menus: Array
  },
  data() {
    return {
      visibility: false,
      axis: {x: 0, y: 0}
    }
  },
  watch: {
    visibility(val) {
      if (!val)
        document.removeEventListener('mousedown', this.close)
    }
  },
  methods: {
    close(e) {
      const el = this.$refs.content

      if (e.clientX >= this.axis.x && e.clientX <= this.axis.x + el.clientWidth &&
        e.clientY >= this.axis.y && e.clientY <= this.axis.y + el.clientHeight) {
        return
      }

      this.visibility = false
    },
    show(e) {
      document.addEventListener('mousedown', this.close)
      this.axis = {x: e.clientX, y: e.clientY}
      this.visibility = true
    },
    onMenuClick(name) {
      this.visibility = false
      this.$emit('click', name)
    }
  }
}
</script>

<style scoped>
  .content {
    position: fixed;
    z-index: 9999;
    background-color: #f9f9f9;
    min-width: 160px;
    box-shadow: 0px 8px 16px 0px rgba(0, 0, 0, 0.2);
  }

  .content a {
    color: black;
    padding: 5px 16px;
    text-decoration: none;
    display: block;
  }

  .content a:hover {
    background-color: #90C8F6;
    cursor: default;
  }
</style>
