import { syncUI } from '../app.js';

export let masterVolume = parseFloat(localStorage.getItem('masterVolume') || '0.5');
export let musicVolume = parseFloat(localStorage.getItem('musicVolume') || '0.5');
