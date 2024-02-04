(ns jepsen.raft
  "Tests for raft"
  (:require [clojure.tools.logging :refer [info]]
            [clojure.string :as str]
            [knossos.model :as model]
            [jepsen [client :as client]
             [cli :as cli]
             [db :as db]
             [tests :as tests]
             [control :as c]
             [checker :as checker]
             [generator :as gen]
             [nemesis :as nemesis]
             [util :refer [parse-long]]]
            [jepsen.checker.timeline :as timeline]
            [jepsen.control.util :as cu]
            [jepsen.os.debian :as debian]))

(def dir "/opt/kv-server")
(def logfile (str dir "/kv-server.log"))
(def pidfile (str dir "/kv-server.pid"))
(def data-dir (str dir "/data"))
(def server-binary "./kv-server")
(def client-binary "./kv-client")
(def raft-port ":5254")
(def kv-port ":5255")
(def bootstrap-node "n1")
(def timeout-msg-pattern
  (re-pattern "operation failed: client-specified timeout elapsed"))

(defn cluster
  "Constructs a cluster string for a test, like
    \"n1:192.168.1.2:8901,n2:192.168.1.3:8901,...\""
  [test port]
  (->> (:nodes test)
       (map (fn [node]
              (str (name node) ":" (name node) port)))
       (str/join ",")))

(defn parse-long-nil
  "Parses a string to a Long. Passes through `nil`."
  [s]
  (when s (parse-long s)))

(defn install!
  "Install raft-example"
  [node version]
  (info node "Installing raft-example" version)
  (debian/install [:git-core])
  (c/exec :mkdir :-p dir)
  (c/cd dir
         (when-not (cu/exists? "raft-example")
           (c/exec :git :clone "https://github.com/jmsadair/raft-example.git")))
  (c/cd dir
        (c/cd "raft-example/cmd/kv-server"
              (c/exec :go :build)
              (c/exec :cp :-f "kv-server" "/opt/kv-server")))
  (c/cd dir
        (c/cd "raft-example/cmd/kv-client"
              (c/exec :go :build)
              (c/exec :cp :-f "kv-client" "/opt/kv-server"))))

(defn bootstrap!
  "Bootstrap a server with an initial configuration"
  [test node]
  (info node "Bootstrapping server")
  (c/cd dir
        (c/exec server-binary :-id node :-d data-dir :bootstrap :-c (cluster test raft-port)
                (c/lit (str ">>" logfile " 2>&1 &")))))

(defn start!
  "Start the server"
  [node]
  (info node "Starting server")
  (c/su
   (cu/start-daemon!
    {:logfile logfile
     :pidfile pidfile
     :chdir   dir}
    "kv-server"
    :-id node
    :-d data-dir
    :start
    :-a kv-port
    :-ra raft-port)
   (Thread/sleep 10000)))

(defn stop!
  "Stop the server"
  [node]
  (info node "Stopping server")
  (cu/stop-daemon! "kv-server" pidfile)
  (c/su (c/exec :rm :-rf dir)))

(defn db
  "Setup and tear down the server"
  [version]
  (reify db/DB
    (setup! [_ test node]
      (info node "Setting up server")
      (install! node version)
      (when (= (name node) bootstrap-node)
        (bootstrap! test node))
      (start! node))
    (teardown! [_ _ node]
      (info node "Tearing down server")
      (stop! node))))

(defn r [_ _] {:type :invoke, :f :read, :value nil})

(defn w [_ _] {:type :invoke, :f :write, :value (rand-int 5)})

(defn server-get!
  "Get a value for a key"
  [test node k]
  (let [value (c/on node
                    (c/cd dir
                          (c/exec client-binary
                                  :-c (cluster test kv-port)
                                  :get
                                  :-k k)))]
    (if (empty? value)
      nil
      value)))

(defn server-put!
  "Set a value for a key"
  [test node k v]
  (c/on node
        (c/su
         (c/cd dir
               (c/exec client-binary
                       :-c (cluster test kv-port)
                       :put
                       :-k k
                       :-v v)))))

(defrecord ServerClient [client]
  client/Client
  (setup! [_ _])
  (open! [this _ node]
    (assoc this :client node))
  (invoke! [_ test op]
    (try
      (case (:f op)
        :read (assoc op :type :ok :value (parse-long-nil (server-get! test client "x")))
        :write (do (server-put! test client "x" (:value op)) (assoc op :type :ok)))
      (catch Exception e
        (let [msg (str/trim (.getMessage e))]
          (cond
            (not (nil? (re-find timeout-msg-pattern msg)))
            (assoc op :type (if (= :read (:f op)) :fail :info) :error :timeout)
            :else (throw e))))))
  (teardown! [_ _])
  (close! [_ _]))

(defn raft-test
  "Takes options from CLI and constructs a test map"
  [opts]
  (merge tests/noop-test
         opts
         {:pure-generators true
          :name            "raft-test"
          :os              debian/os
          :db              (db "v0.0.1")
          :client          (ServerClient. nil)
          :nemesis         (nemesis/partition-random-halves)
          :checker (checker/compose
                    {:perf (checker/perf)
                     :timeline (timeline/html)
                     :linear (checker/linearizable {:model (model/cas-register) :algorithm :linear})})
          :generator       (->> (gen/mix [r w])
                                (gen/stagger 1/100)
                                (gen/nemesis (cycle [(gen/sleep 5) {:type :info, :f :start} (gen/sleep 5) {:type :info, :f :stop}]))
                                (gen/time-limit (:time-limit opts)))}))

(defn -main
  "Handles CLI args"
  [& args]
  (cli/run! (cli/single-test-cmd {:test-fn raft-test})
            args))
