
;; Storage.

;; Write.
(def! sstore (+ 2 2))

;; Read.
(def! sload (+ 2 2))

;; (def! sload (+ 2 2))

(def! contracts.register (lambda (name address)
                           (def! (sym name) address)))

