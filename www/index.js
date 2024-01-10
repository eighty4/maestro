function registerIntersectionObserver(color) {
    const cardSection = document.getElementById(color + '-fg-section')
    const card = cardSection.querySelector(`#${color}-card`)
    const miniCircle = cardSection.querySelector(`#mini-${color}`)
    const fgCircle = cardSection.querySelector(`#big-${color}-fg`)

    function updateMinimizedOffset() {
        console.log(color, 'updateMinimizedOffset')
        const minimizedOffset = getContainerOffset(miniCircle, card.parentElement)
        cardSection.style.setProperty('--minimized-circle-top', `${minimizedOffset.top}px`)
        cardSection.style.setProperty('--minimized-circle-left', `${minimizedOffset.left}px`)
    }

    updateMinimizedOffset()
    window.addEventListener('resize', updateMinimizedOffset, {passive: true})

    let initialScrollY = null

    function setMinimizeProgress(num) {
        cardSection.style.setProperty('--minimize-progress', `${num}`)
    }

    function updateScrollOffset(e) {
        console.log(color, 'updateScrollOffset', 'initial', initialScrollY, 'current', window.scrollY)
        if (window.scrollY < initialScrollY) {
            setMinimizeProgress(0)
        } else {
            const diffScrollY = window.scrollY - initialScrollY
            setMinimizeProgress(diffScrollY / card.getBoundingClientRect().height)
        }
    }

    function intersectionObserverCallback(entries) {
        const {intersectionRatio} = entries[0]
        console.log(color, 'intersectionObserverCallback', intersectionRatio)
        if (intersectionRatio === 1) {
            window.removeEventListener('scroll', updateScrollOffset)
            setMinimizeProgress(1)
            fgCircle.classList.remove('minimizing')
            fgCircle.classList.add('minimized')
            observer.unobserve(card)
            window.removeEventListener('resize', updateMinimizedOffset)
        } else if (intersectionRatio > 0) {
            fgCircle.classList.add('minimizing')
            initialScrollY = window.scrollY
            window.addEventListener('scroll', updateScrollOffset, {passive: true})
        } else if (intersectionRatio === 0 && initialScrollY !== null) {
            fgCircle.classList.remove('minimizing')
            window.removeEventListener('scroll', updateScrollOffset)
            initialScrollY = null
        }
    }

    const observer = new IntersectionObserver(intersectionObserverCallback, {
        threshold: [0, 0.0000000001, 1],
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
