function registerIntersectionObserver(color) {
    const cardSection = document.getElementById(color + '-fg-section')
    const card = cardSection.querySelector(`#${color}-card`)
    const miniCircle = cardSection.querySelector(`#mini-${color}`)
    const fgCircle = cardSection.querySelector(`#big-${color}-fg`)
    function updateMinimizedOffset() {
        const minimizedOffset = getContainerOffset(miniCircle, card.parentElement)
        card.parentElement.style.setProperty('--minimized-circle-top', `${minimizedOffset.top}px`)
        card.parentElement.style.setProperty('--minimized-circle-left', `${minimizedOffset.left}px`)
    }
    updateMinimizedOffset()
    window.onresize = updateMinimizedOffset

    function intersectionObserverCallback(entries) {
        if (entries[0].intersectionRatio >.5) {
            fgCircle.classList.add('minimize')
            observer.unobserve(card)
            window.onresize = undefined
        }
    }

    const observer = new IntersectionObserver(intersectionObserverCallback, {
        threshold: [.5],
    })
    observer.observe(card)
}

document.addEventListener('DOMContentLoaded', () => {
    registerIntersectionObserver('red')
    registerIntersectionObserver('yellow')
    registerIntersectionObserver('green')
})

function getContainerOffset(child, parent) {
    const childOffset = getDocumentOffset(child)
    const parentOffset = getDocumentOffset(parent)
    return {
        top: childOffset.top - parentOffset.top,
        left: childOffset.left - parentOffset.left,
    }
}

function getDocumentOffset(elem) {
    const box = elem.getBoundingClientRect()
    const scrollTop = document.documentElement.scrollTop || document.body.scrollTop
    const scrollLeft = document.documentElement.scrollLeft || document.body.scrollLeft
    const clientTop = document.documentElement.clientTop || document.body.clientTop || 0
    const clientLeft = document.documentElement.clientLeft || document.body.clientLeft || 0
    const top = box.top + scrollTop - clientTop
    const left = box.left + scrollLeft - clientLeft
    return {top: Math.round(top), left: Math.round(left)}
}
