; Independant sources for different random numbers
(define source (unit/gen))
(define source2 (unit/gen))
(define source3 (unit/gen))
(define source4 (unit/gen))

; Latches for freezing outputs of the unit/gens
(define latch (unit/latch))
(define latch2 (unit/latch))
(define latch3 (unit/latch))
(define latch4 (unit/latch))

; Predeclare slope. We'll use its "eoc" output to feedback a trigger signal at the end of each note to cause new random
; values to be locked into the latches and be outputted. Round and round we go.
(define slope (unit/slope))

(-> latch (table :in (<- source :noise) :trigger (<- slope :eoc)))
(-> latch2 (table :in (<- source2 :noise) :trigger (<- slope :eoc)))
(-> latch3 (table :in (<- source3 :noise) :trigger (<- slope :eoc)))
(-> latch4 (table :in (<- source4 :noise) :trigger (<- slope :eoc)))

; Value ranges for parameters
(define rise (unit/mult))
(define fall (unit/mult))
(define cutoff (unit/mult))
(define left-compress (unit/dynamics))
(define right-compress (unit/dynamics))

(-> rise (table :x (ms 500) :y (<- latch)))
(-> fall (table :x (ms 3000) :y (<- latch2)))
(-> cutoff (table :x (hz 1000) :y (<- latch3)))

(define quantize (unit/quantize))
(define gate (unit/gate))
(define reverb (unit/reverb))

(-> quantize 
    (table :in (<- latch4)
           :tonic (hz "C2")
           :intervals (list (theory/interval :perfect 1)
                            (theory/interval :minor 2)
                            (theory/interval :minor 3)
                            (theory/interval :perfect 5)
                            (theory/interval :minor 7))))

(-> source (table :freq (<- quantize)))

; Slope will cycle and keep emitting "eoc" triggers which are triggering the latches.
(-> slope
    (table :rise (<- rise)
           :fall (<- fall)
           :cycle mode/on
           :trigger 1))

(define mix (unit/mix (table :size 2)))

(-> mix 
    (list (table :in (<- source :saw))
          (table :in (<- source :sub-pulse) :level (db -9))))

(-> gate
    (table :in (<- mix)
           :control (<- slope)
           :cutoff-high (<- cutoff)))

(-> reverb (table :a (<- gate) 
                  :b (<- gate) 
                  :decay 0.8
                  :cutoff-pre (hz 300)
                  :cutoff-post (hz 600)))

(let
  ((compression (table :above 0.1 :threshold (db -6) :relax (ms 300)))
   (left (<- reverb :a))
   (right (<- reverb :b)))
  (-> left-compress (table-merge compression (table :in left :control left)))
  (-> right-compress (table-merge compression (table :in right :control right))))

(emit (<- left-compress) (<- right-compress))
