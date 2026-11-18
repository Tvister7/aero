export function focusField(id: string) {
  const field = document.getElementById(id)
  if (!field) return
  field.scrollIntoView({ behavior: 'smooth', block: 'center' })
  field.focus()
}
