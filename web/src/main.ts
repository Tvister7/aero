import { mount } from 'svelte'
import './app.css'
import App from './App.svelte'
import { bindTelegram } from './telegram'

bindTelegram()

const target = document.getElementById('app')
if (!target) throw new Error('Не найден корень приложения')

const app = mount(App, { target })

export default app
