<template>
  <div class="text">
    <span>Oops...</span>
    <p>{{ message }}</p>
  </div>
</template>

<script>
export default {
  name: 'Error',
  props: {
    err: String
  },
  computed: {
    message() {
      const err = this.err

      if (err === 'offline')
        return this.$t('Device offline')
      else if (err === 'full')
        return this.$t('Sessions is full')
      return ''
    }
  }
}
</script>

<style scoped>
.text {
  font: 400 3vmin "Courgette";
  text-align: center;
  color: red;
}

.text span {
  font-size: 8vmin;
  margin-bottom: 10px;
}
</style>
