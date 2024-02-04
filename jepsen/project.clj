(defproject jepsen.raft "0.1.0-SNAPSHOT"
  :description "A Jepsen test for raft"
  :url "http://github.com/jmsadair/raft-example"
  :license {:name "The MIT License"
            :url  "http://opensource.org/licenses/MIT"}
  :main jepsen.raft
  :dependencies [[org.clojure/clojure "1.10.0"]
                 [jepsen "0.2.1-SNAPSHOT"]])
