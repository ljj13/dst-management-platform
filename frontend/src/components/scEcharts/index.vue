<template>
  <div
    ref="scEcharts"
    :style="{height:height, width:width}"
  />
</template>

<script>
import * as echarts from 'echarts'
import T from './echarts-theme-T.js'

echarts.registerTheme('T', T)

const unwarp = obj => obj && (obj.__v_raw || obj.valueOf() || obj)

export default {
  ...echarts,
  name: "ScEcharts",
  props: {
    height: { type: String, default: "100%" },
    width: { type: String, default: "100%" },
    nodata: { type: Boolean, default: false },
    option: { type: Object, default: () => {} },
  },
  data() {
    return {
      isActivat: false,
      myChart: null,
      resizeHandler: null,
    }
  },
  computed: {
    myOptions: function() {
      return this.option || {}
    },
  },
  watch: {
    option: {
      deep: true,
      handler (v) {
        if (this.myChart) {
          unwarp(this.myChart).setOption(v)
        }
      },
    },
  },
  activated(){
    if(!this.isActivat){
      this.$nextTick(() => {
        if (this.myChart) {
          this.myChart.resize()
        }
      })
    }
  },
  deactivated(){
    this.isActivat = false
  },
  beforeUnmount() {
    if (this.resizeHandler) {
      window.removeEventListener('resize', this.resizeHandler)
      this.resizeHandler = null
    }
    if (this.myChart) {
      this.myChart.dispose()
      this.myChart = null
    }
  },
  mounted(){
    this.isActivat = true
    this.$nextTick(() => {
      this.draw()
    })
  },
  methods: {
    draw(){
      var myChart = echarts.init(this.$refs.scEcharts, 'T', { renderer: 'svg' })
      myChart.setOption(this.myOptions)
      this.myChart = myChart
      this.resizeHandler = () => {
        if (this.myChart) {
          this.myChart.resize()
        }
      }
      window.addEventListener('resize', this.resizeHandler)
    },
    redraw(option) {
      if (this.myChart) {
        this.myChart.dispose()
      }
      var myChart = echarts.init(this.$refs.scEcharts, 'T', { renderer: 'svg' })
      myChart.setOption(option)
      this.myChart = myChart
    },
    setOption(option) {
      if (this.myChart) {
        this.myChart.setOption(option)
      }
    },
  },
}
</script>
