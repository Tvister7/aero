const nameMessage = 'Имя может содержать буквы, пробел, дефис и апостроф'
const phoneCharsMessage = 'Телефон может содержать цифры, пробел, +, скобки и дефис'
const phoneDigitsMessage = 'В телефоне должно быть не меньше 9 цифр'

export function nameError(value: string): string {
  const name = value.trim()
  if (!name) return ''
  let letters = false
  for (const char of name) {
    if (/\p{L}/u.test(char)) {
      letters = true
      continue
    }
    if (char === ' ' || char === '-' || char === "'" || char === '’') continue
    return nameMessage
  }
  return letters ? '' : nameMessage
}

export function phoneError(value: string): string {
  const phone = value.trim()
  if (!phone) return ''
  let digits = 0
  for (const char of phone) {
    if (char >= '0' && char <= '9') {
      digits += 1
      continue
    }
    if (char === ' ' || char === '+' || char === '-' || char === '(' || char === ')') continue
    return phoneCharsMessage
  }
  return digits >= 9 ? '' : phoneDigitsMessage
}
