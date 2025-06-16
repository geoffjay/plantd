/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./views/**/*.templ",
    "./views/**/*.go", 
    "./static/**/*.html"
  ],
  theme: {
    extend: {
      colors: {
        plantd: {
          primary: '#10B981',
          secondary: '#06B6D4',
          accent: '#8B5CF6',
          danger: '#EF4444',
          warning: '#F59E0B',
          success: '#10B981'
        }
      }
    }
  },
  plugins: []
}
