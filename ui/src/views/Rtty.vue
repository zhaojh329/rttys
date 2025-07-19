<template>
  <RttyTerm :devid="devid"/>
</template>

<script setup>
import RttyTerm from '../components/RttyTerm.vue'

defineProps({
  devid: {
    type: String,
    required: true
  }
})

</script>
