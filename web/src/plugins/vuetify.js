import 'vuetify/styles'
import '@mdi/font/css/materialdesignicons.css'
import { createVuetify } from 'vuetify'

export default createVuetify({
  theme: {
    defaultTheme: localStorage.getItem('telm-theme') || 'dark',
    themes: {
      dark: {
        dark: true,
        colors: {
          background:        '#090e1a',
          surface:           '#0f1623',
          'surface-variant': '#1a2235',
          'surface-bright':  '#212d42',
          primary:           '#6366f1',
          secondary:         '#8b5cf6',
          error:             '#ef4444',
          warning:           '#f59e0b',
          info:              '#3b82f6',
          success:           '#10b981',
          'on-surface':      '#cbd5e1',
          'on-background':   '#cbd5e1',
        },
      },
      light: {
        dark: false,
        colors: {
          background:        '#f1f5f9',
          surface:           '#ffffff',
          'surface-variant': '#f8fafc',
          'surface-bright':  '#f1f5f9',
          primary:           '#6366f1',
          secondary:         '#8b5cf6',
          error:             '#dc2626',
          warning:           '#d97706',
          info:              '#2563eb',
          success:           '#059669',
          'on-surface':      '#0f172a',
          'on-background':   '#0f172a',
        },
      },
    },
  },
  defaults: {
    VCard:      { rounded: 'lg', border: true, elevation: 0 },
    VBtn:       { rounded: 'md' },
    VTextField: { variant: 'outlined', density: 'compact', color: 'primary' },
    VSelect:    { variant: 'outlined', density: 'compact', color: 'primary' },
    VChip:      { rounded: 'md', size: 'x-small' },
    VTable:     { density: 'compact' },
  },
})
