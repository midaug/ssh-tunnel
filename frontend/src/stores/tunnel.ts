import {defineStore} from 'pinia'
import {ref} from 'vue'
import * as App from '../../wailsjs/go/main/App'
import {config} from '../../wailsjs/go/models'

export type Tunnel = config.Tunnel

export const useTunnelStore = defineStore('tunnel', () => {
  const tunnels = ref<Tunnel[]>([])
  const loading = ref(false)

  async function load() {
    loading.value = true
    try {
      tunnels.value = await App.TunnelList()
    } finally {
      loading.value = false
    }
  }

  function onStatus(id: string, status: string, lastError: string) {
    const t = tunnels.value.find(x => x.id === id)
    if (t) {
      t.status = status
      t.lastError = lastError
    }
  }

  async function start(id: string) {
    await App.TunnelStart(id)
  }
  async function stop(id: string) {
    await App.TunnelStop(id)
  }
  async function restart(id: string) {
    await App.TunnelRestart(id)
  }
  async function remove(id: string) {
    await App.TunnelDelete(id)
    await load()
  }
  async function save(t: Tunnel): Promise<Tunnel> {
    const saved = await App.TunnelSave(t)
    await load()
    return saved
  }

  return {tunnels, loading, load, onStatus, start, stop, restart, remove, save}
})
