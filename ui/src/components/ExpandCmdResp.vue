<template>
  <div>
    <p>{{ $t('Stdout') + ':' }}</p>
    <Input type="textarea" :value="stdout" readonly :autosize="{minRows: 1, maxRows: 20}"/>
    <p style="margin-top: 10px">{{ $t('Stderr') + ':' }}</p>
    <Input type="textarea" :value="stderr" readonly :autosize="{minRows: 1, maxRows: 20}"/>
  </div>
</template>

<script>
export default {
  name: 'ExpandCmdResp',
  props: {
    stdout: String,
    stderr: String
  }
}
</script>
